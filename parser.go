package juice

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

// ConfigurationParser is the interface for parsing configuration.
type ConfigurationParser interface {
	// Parse parses the configuration from the reader.
	Parse(reader io.Reader) (*Configuration, error)
}

// XMLParser is the parser for XML configuration.
type XMLParser struct {
	configuration Configuration
	FS            fs.FS
}

// Parse implements ConfigurationParser.
func (p XMLParser) Parse(reader io.Reader) (*Configuration, error) {
	decoder := xml.NewDecoder(reader)
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			switch token.Name.Local {
			case "environments":
				envs, err := p.parseEnvironments(decoder, token)
				if err != nil {
					return nil, err
				}
				p.configuration.Environments = *envs
			case "mappers":
				var mappers = Mappers{cfg: &p.configuration}
				if err := p.parseMappers(&mappers, decoder); err != nil {
					return nil, err
				}
				p.configuration.Mappers = mappers
			case "settings":
				settings, err := p.parseSettings(decoder)
				if err != nil {
					return nil, err
				}
				p.configuration.Settings = *settings
			}
		}
	}
	return &p.configuration, nil
}

func (p XMLParser) parseEnvironments(decoder *xml.Decoder, token xml.StartElement) (*Environments, error) {
	var envs Environments
	for _, attr := range token.Attr {
		if attr.Name.Local == "default" {
			envs.Default = attr.Value
			break
		}
	}
	if envs.Default == "" {
		return nil, errors.New("default environment is required")
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			if token.Name.Local == "environment" {
				environment, err := p.parseEnvironment(decoder, token)
				if err != nil {
					return nil, err
				}
				if _, exists := envs.envs[environment.ID()]; exists {
					return nil, fmt.Errorf("duplicate environment id: %s", environment.ID())
				}
				if envs.envs == nil {
					envs.envs = make(map[string]*Environment)
				}
				envs.envs[environment.ID()] = environment
			}
		case xml.EndElement:
			if token.Name.Local == "environments" {
				return &envs, nil
			}
		}
	}
	return &envs, nil
}

func (p XMLParser) parseEnvironment(decoder *xml.Decoder, token xml.StartElement) (*Environment, error) {
	var env = &Environment{}
	for _, attr := range token.Attr {
		env.setAttr(attr.Name.Local, attr.Value)
	}
	if env.ID() == "" {
		return nil, errors.New("environment id is required")
	}
	provider := env.provider()
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			switch token.Name.Local {
			case "dataSource":
				env.DataSource, err = p.parseDataSource(decoder, provider)
				if err != nil {
					return nil, err
				}
			case "driver":
				env.Driver, err = p.parseDriver(decoder, provider)
				if err != nil {
					return nil, err
				}
			case "maxIdleConnNum":
				env.MaxIdleConnNum, err = p.parseMaxIdleConnNum(decoder)
				if err != nil {
					return nil, err
				}
			case "maxOpenConnNum":
				env.MaxOpenConnNum, err = p.parseMaxOpenConnNum(decoder, provider)
				if err != nil {
					return nil, err
				}
			case "maxConnLifetime":
				env.MaxConnLifetime, err = p.parseMaxConnLifetime(decoder, provider)
				if err != nil {
					return nil, err
				}
			case "maxIdleConnLifetime":
				env.MaxIdleConnLifetime, err = p.parseMaxIdleConnLifetime(decoder, provider)
				if err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if token.Name.Local == "environment" {
				return env, nil
			}
		}
	}
	return env, nil
}

func (p XMLParser) parseMappers(mappers *Mappers, decoder *xml.Decoder) error {
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		switch token := token.(type) {
		case xml.StartElement:
			if token.Name.Local == "mapper" {
				mapper, err := p.parseMapper(decoder, token)
				if err != nil {
					return err
				}
				for key, stmt := range mapper.statements {
					if err = mappers.setStatementByID(key, stmt); err != nil {
						return err
					}
				}
				mapper.mappers = mappers
			}
		case xml.EndElement:
			if token.Name.Local == "mappers" {
				return nil
			}
		}
	}
	return nil
}

func (p XMLParser) parseMapper(decoder *xml.Decoder, token xml.StartElement) (*Mapper, error) {
	mapper := &Mapper{}
	for _, attr := range token.Attr {
		mapper.setAttribute(attr.Name.Local, attr.Value)
	}

	if mapper.resource = mapper.Attribute("resource"); mapper.resource != "" {
		return p.parseMapperByResource(mapper.resource)
	}

	if mapper.url = mapper.Attribute("url"); mapper.url != "" {
		return p.parseMapperByURL(mapper.url)
	}

	if mapper.namespace = mapper.Attribute("namespace"); mapper.namespace == "" {
		return nil, errors.New("mapper requires namespace attribute")
	}

	mapper.statements = make(map[string]*Statement)

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			action := Action(token.Name.Local)
			switch action {
			case Select, Insert, Update, Delete:
				stmt := &Statement{action: action, mapper: mapper}
				if err = p.parseStatement(stmt, decoder, token); err != nil {
					return nil, err
				}
				key := stmt.Key()
				if _, exists := mapper.statements[key]; exists {
					return nil, fmt.Errorf("duplicate statement id: %s", stmt.ID())
				}
				mapper.statements[key] = stmt
			case "sql":
				sqlNode := &SQLNode{mapper: mapper}
				if err = p.parseSQLNode(sqlNode, decoder, token); err != nil {
					return nil, err
				}
				if err = mapper.setSqlNode(sqlNode); err != nil {
					return nil, err
				}
			case "resultMap":
				resultMap, err := p.parseResultMap(decoder, token)
				if err != nil {
					return nil, err
				}
				if err := resultMap.init(); err != nil {
					return nil, err
				}
				if err = mapper.setResultMap(resultMap); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if token.Name.Local == "mapper" {
				return mapper, nil
			}
		}
	}
	return mapper, nil
}

func (p XMLParser) parseMapperByReader(reader io.Reader) (mapper *Mapper, err error) {
	decoder := xml.NewDecoder(reader)
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			if token.Name.Local == "mapper" {
				if mapper, err = p.parseMapper(decoder, token); err != nil {
					return nil, err
				}
				break
			}
		}
	}
	return mapper, err
}

func (p XMLParser) parseMapperByResource(resource string) (*Mapper, error) {
	var (
		reader io.ReadCloser
		err    error
	)
	reader, err = p.FS.Open(resource)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	return p.parseMapperByReader(reader)
}

func (p XMLParser) parseMapperByHttpResponse(url string) (*Mapper, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return p.parseMapperByReader(resp.Body)
}

func (p XMLParser) parseMapperByURL(url string) (*Mapper, error) {
	// prepare url schema
	items := strings.Split(url, "://")
	if len(items) != 2 {
		return nil, fmt.Errorf("invalid url: %s", url)
	}
	schema := items[0]
	switch schema {
	case "file":
		return p.parseMapperByResource(items[1])
	case "http", "https":
		return p.parseMapperByHttpResponse(url)
	default:
		return nil, errors.New("invalid url schema")
	}
}

func (p XMLParser) parseStatement(stmt *Statement, decoder *xml.Decoder, token xml.StartElement) error {
	for _, attr := range token.Attr {
		stmt.setAttribute(attr.Name.Local, attr.Value)
	}
	if stmt.ID() == "" {
		return fmt.Errorf("%s statement id is required", stmt.Action())
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		switch token := token.(type) {
		case xml.StartElement:
			node, err := p.parseTags(stmt.Mapper(), decoder, token)
			if err != nil {
				return err
			}
			stmt.Nodes = append(stmt.Nodes, node)
		case xml.CharData:
			text := string(token)
			if char := strings.TrimSpace(text); char != "" {
				node := TextNode(char)
				stmt.Nodes = append(stmt.Nodes, node)
			}
		case xml.EndElement:
			switch token.Name.Local {
			case stmt.action.String():
				return nil
			default:
				return fmt.Errorf("unexpected end element: %s", token.Name.Local)
			}
		}
	}
	return nil
}

func (p XMLParser) parseTags(mapper *Mapper, decoder *xml.Decoder, token xml.StartElement) (Node, error) {
	switch token.Name.Local {
	case "if":
		return p.parseIf(mapper, decoder, token)
	case "where":
		return p.parseWhere(mapper, decoder)
	case "trim":
		return p.parseTrim(mapper, decoder, token)
	case "foreach":
		return p.parseForeach(mapper, decoder, token)
	case "set":
		return p.parseSet(mapper, decoder)
	case "include":
		return p.parseInclude(mapper, decoder, token)
	case "choose":
		return p.parseChoose(mapper, decoder)
	}
	return nil, fmt.Errorf("unknown tag: %s", token.Name.Local)
}

func (p XMLParser) parseInclude(mapper *Mapper, decoder *xml.Decoder, token xml.StartElement) (Node, error) {
	var ref string
	for _, attr := range token.Attr {
		switch attr.Name.Local {
		case "refid":
			ref = attr.Value
		}
	}
	if ref == "" {
		return nil, errors.New("include ref is required")
	}

	includeNode := &IncludeNode{RefId: ref, mapper: mapper}

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.EndElement:
			if token.Name.Local == "include" {
				return includeNode, nil
			}
		}
	}
	return includeNode, nil
}

func (p XMLParser) parseSet(mapper *Mapper, decoder *xml.Decoder) (Node, error) {
	setNode := &SetNode{}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			node, err := p.parseTags(mapper, decoder, token)
			if err != nil {
				return nil, err
			}
			setNode.Nodes = append(setNode.Nodes, node)
		case xml.CharData:
			text := string(token)
			if char := strings.TrimSpace(text); char != "" {
				node := TextNode(char)
				setNode.Nodes = append(setNode.Nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "set" {
				return setNode, nil
			}
		}
	}
	return nil, nil
}

func (p XMLParser) parseIf(mapper *Mapper, decoder *xml.Decoder, token xml.StartElement) (Node, error) {
	ifNode := &IfNode{}
	var test string
	for _, attr := range token.Attr {
		if attr.Name.Local == "test" {
			test = attr.Value
			break
		}
	}
	if test == "" {
		return nil, errors.New("if node requires test attribute")
	}

	// parse condition expression
	if err := ifNode.Parse(test); err != nil {
		return nil, err
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			node, err := p.parseTags(mapper, decoder, token)
			if err != nil {
				return nil, err
			}
			ifNode.Nodes = append(ifNode.Nodes, node)
		case xml.CharData:
			text := string(token)
			if char := strings.TrimSpace(text); char != "" {
				node := TextNode(char)
				ifNode.Nodes = append(ifNode.Nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "if" {
				return ifNode, nil
			}
		}
	}
	return ifNode, nil
}

func (p XMLParser) parseWhere(mapper *Mapper, decoder *xml.Decoder) (Node, error) {
	whereNode := &WhereNode{}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			node, err := p.parseTags(mapper, decoder, token)
			if err != nil {
				return nil, err
			}
			whereNode.Nodes = append(whereNode.Nodes, node)
		case xml.CharData:
			text := string(token)
			if char := strings.TrimSpace(text); char != "" {
				node := TextNode(char)
				whereNode.Nodes = append(whereNode.Nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "where" {
				return whereNode, nil
			}
		}
	}
	return whereNode, nil
}

func (p XMLParser) parseTrim(mapper *Mapper, decoder *xml.Decoder, token xml.StartElement) (Node, error) {
	trimNode := &TrimNode{}
	for _, attr := range token.Attr {
		switch attr.Name.Local {
		case "prefix":
			trimNode.Prefix = attr.Value
		case "prefixOverrides":
			trimNode.PrefixOverrides = attr.Value
		case "suffix":
			trimNode.Suffix = attr.Value
		case "suffixOverrides":
			trimNode.SuffixOverrides = attr.Value
		}
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			node, err := p.parseTags(mapper, decoder, token)
			if err != nil {
				return nil, err
			}
			trimNode.Nodes = append(trimNode.Nodes, node)
		case xml.EndElement:
			if token.Name.Local == "trim" {
				return trimNode, nil
			}
		}
	}
	return trimNode, nil
}

func (p XMLParser) parseForeach(mapper *Mapper, decoder *xml.Decoder, token xml.StartElement) (Node, error) {
	foreachNode := &ForeachNode{}
	for _, attr := range token.Attr {
		switch attr.Name.Local {
		case "collection":
			foreachNode.Collection = attr.Value
		case "item":
			foreachNode.Item = attr.Value
		case "index":
			foreachNode.Index = attr.Value
		case "open":
			foreachNode.Open = attr.Value
		case "separator":
			foreachNode.Separator = attr.Value
		case "close":
			foreachNode.Close = attr.Value
		}
	}
	if foreachNode.Collection == "" {
		foreachNode.Collection = defaultParamKey
	}
	if foreachNode.Item == "" {
		return nil, errors.New("foreach node requires item attribute")
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			node, err := p.parseTags(mapper, decoder, token)
			if err != nil {
				return nil, err
			}
			foreachNode.Nodes = append(foreachNode.Nodes, node)
		case xml.CharData:
			text := string(token)
			if char := strings.TrimSpace(text); char != "" {
				node := TextNode(char)
				foreachNode.Nodes = append(foreachNode.Nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "foreach" {
				return foreachNode, nil
			}
		}
	}
	return foreachNode, nil
}

func (p XMLParser) parseChoose(mapper *Mapper, decoder *xml.Decoder) (Node, error) {
	chooseNode := &ChooseNode{}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			switch token.Name.Local {
			case "when":
				node, err := p.parseWhen(mapper, decoder, token)
				if err != nil {
					return nil, err
				}
				chooseNode.WhenNodes = append(chooseNode.WhenNodes, node)
			case "otherwise":
				if chooseNode.OtherwiseNode != nil {
					return nil, errors.New("otherwise is only once")
				}
				node, err := p.parseOtherwise(mapper, decoder)
				if err != nil {
					return nil, err
				}
				chooseNode.OtherwiseNode = node
			}

		case xml.EndElement:
			if token.Name.Local == "choose" {
				return chooseNode, nil
			}
		}
	}
	return chooseNode, nil
}

func (p XMLParser) parseCharData(decoder *xml.Decoder, endElementName string) (string, error) {
	var charData string
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		switch token := token.(type) {
		case xml.CharData:
			charData = string(token)
		case xml.EndElement:
			if token.Name.Local == endElementName {
				return charData, nil
			}
		}
	}
	return charData, nil
}

func (p XMLParser) parseIntCharData(decoder *xml.Decoder, endElementName string) (int, error) {
	charData, err := p.parseCharData(decoder, endElementName)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(charData)
}

func (p XMLParser) parseMaxIdleConnNum(decoder *xml.Decoder) (int, error) {
	return p.parseIntCharData(decoder, "maxIdleConnNum")
}

func (p XMLParser) parseEnvString(key string, decoder *xml.Decoder, provider EnvValueProvider) (string, error) {
	value, err := p.parseCharData(decoder, key)
	if err != nil {
		return "", err
	}
	return provider.Get(value)
}

func (p XMLParser) parseEnvInt(key string, decoder *xml.Decoder, provider EnvValueProvider) (int, error) {
	value, err := p.parseCharData(decoder, key)
	if err != nil {
		return 0, err
	}
	str, err := provider.Get(value)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(str)
}

func (p XMLParser) parseDataSource(decoder *xml.Decoder, provider EnvValueProvider) (string, error) {
	return p.parseEnvString("dataSource", decoder, provider)
}

func (p XMLParser) parseDriver(decoder *xml.Decoder, provider EnvValueProvider) (string, error) {
	return p.parseEnvString("driver", decoder, provider)
}

func (p XMLParser) parseMaxOpenConnNum(decoder *xml.Decoder, provider EnvValueProvider) (int, error) {
	return p.parseEnvInt("maxOpenConnNum", decoder, provider)
}

func (p XMLParser) parseMaxConnLifetime(decoder *xml.Decoder, provider EnvValueProvider) (int, error) {
	return p.parseEnvInt("maxConnLifetime", decoder, provider)
}

func (p XMLParser) parseMaxIdleConnLifetime(decoder *xml.Decoder, provider EnvValueProvider) (int, error) {
	return p.parseEnvInt("maxIdleConnLifetime", decoder, provider)
}

func (p XMLParser) parseSettings(decoder *xml.Decoder) (*Settings, error) {
	var setting Settings
	if err := decoder.DecodeElement(&setting, nil); err != nil {
		return nil, err
	}
	return &setting, nil
}

func (p XMLParser) parseSQLNode(sqlNode *SQLNode, decoder *xml.Decoder, token xml.StartElement) error {
	for _, attr := range token.Attr {
		if attr.Name.Local == "id" {
			sqlNode.id = attr.Value
			break
		}
	}
	if sqlNode.id == "" {
		return errors.New("sql node requires id attribute")
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		switch token := token.(type) {
		case xml.StartElement:
			tags, err := p.parseTags(sqlNode.mapper, decoder, token)
			if err != nil {
				return err
			}
			sqlNode.nodes = append(sqlNode.nodes, tags)
		case xml.CharData:
			text := string(token)
			if char := strings.TrimSpace(text); char != "" {
				node := TextNode(char)
				sqlNode.nodes = append(sqlNode.nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "sql" {
				return nil
			}
		}
	}
	return nil
}

func (p XMLParser) parseWhen(mapper *Mapper, decoder *xml.Decoder, token xml.StartElement) (Node, error) {
	whenNode := &WhenNode{}
	var test string
	for _, attr := range token.Attr {
		if attr.Name.Local == "test" {
			test = attr.Value
			break
		}
	}
	if test == "" {
		return nil, errors.New("when node require test attribute")
	}

	// parse condition expression
	if err := whenNode.Parse(test); err != nil {
		return nil, err
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			node, err := p.parseTags(mapper, decoder, token)
			if err != nil {
				return nil, err
			}
			whenNode.Nodes = append(whenNode.Nodes, node)
		case xml.CharData:
			text := string(token)
			if char := strings.TrimSpace(text); char != "" {
				node := TextNode(char)
				whenNode.Nodes = append(whenNode.Nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "when" {
				return whenNode, nil
			}
		}
	}
	return whenNode, nil
}

func (p XMLParser) parseOtherwise(mapper *Mapper, decoder *xml.Decoder) (Node, error) {
	otherwiseNode := &OtherwiseNode{}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			tags, err := p.parseTags(mapper, decoder, token)
			if err != nil {
				return nil, err
			}
			otherwiseNode.Nodes = append(otherwiseNode.Nodes, tags)
		case xml.CharData:
			text := string(token)
			if char := strings.TrimSpace(text); char != "" {
				node := TextNode(char)
				otherwiseNode.Nodes = append(otherwiseNode.Nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "otherwise" {
				return otherwiseNode, nil
			}
		}
	}
	return otherwiseNode, nil
}

func (p XMLParser) parseResultMap(decoder *xml.Decoder, token xml.StartElement) (*resultMapNode, error) {
	resultMap := &resultMapNode{}
	for _, attr := range token.Attr {
		if attr.Name.Local == "id" {
			resultMap.id = attr.Value
			break
		}
	}
	if resultMap.id == "" {
		return nil, errors.New("resultMap node requires id attribute")
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			switch token.Name.Local {
			case "id":
				pk, err := p.parseResult(token)
				if err != nil {
					return nil, err
				}
				resultMap.pk = pk
			case "result":
				result, err := p.parseResult(token)
				if err != nil {
					return nil, err
				}
				resultMap.results = append(resultMap.results, result)
			case "association":
				association, err := p.parseAssociation(decoder, token)
				if err != nil {
					return nil, err
				}
				resultMap.associations = append(resultMap.associations, association)
			case "collection":
				coll, err := p.parseCollection(resultMap, decoder, token)
				if err != nil {
					return nil, err
				}
				resultMap.collectionGroup = append(resultMap.collectionGroup, coll)
			}
		case xml.EndElement:
			if token.Name.Local == "resultMap" {
				return resultMap, nil
			}
		}
	}
	return resultMap, nil
}

func (p XMLParser) parseResult(token xml.StartElement) (*result, error) {
	result := &result{}
	for _, attr := range token.Attr {
		switch attr.Name.Local {
		case "column":
			result.column = attr.Value
		case "property":
			result.property = attr.Value
		}
	}
	if result.column == "" {
		return nil, errors.New("result node requires column attribute")
	}
	if result.property == "" {
		return nil, errors.New("result node requires property attribute")
	}
	return result, nil
}

func (p XMLParser) parseAssociation(decoder *xml.Decoder, token xml.StartElement) (*association, error) {
	association := &association{}
	for _, attr := range token.Attr {
		switch attr.Name.Local {
		case "property":
			association.property = attr.Value
		}
	}
	if association.property == "" {
		return nil, errors.New("association node requires property attribute")
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			switch token.Name.Local {
			case "result":
				result, err := p.parseResult(token)
				if err != nil {
					return nil, err
				}
				association.results = append(association.results, result)
			case "association":
				newAssociation, err := p.parseAssociation(decoder, token)
				if err != nil {
					return nil, err
				}
				association.associations = append(association.associations, newAssociation)
			}
		case xml.EndElement:
			if token.Name.Local == "association" {
				return association, nil
			}
		}
	}
	return association, nil
}

func (p XMLParser) parseCollection(parent primaryResult, decoder *xml.Decoder, token xml.StartElement) (*collection, error) {
	coll := &collection{}
	for _, attr := range token.Attr {
		switch attr.Name.Local {
		case "property":
			coll.property = attr.Value
		}
	}
	if coll.property == "" {
		return nil, errors.New("collection node requires property attribute")
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			switch token.Name.Local {
			case "result":
				result, err := p.parseResult(token)
				if err != nil {
					return nil, err
				}
				coll.resultGroup = append(coll.resultGroup, result)
			case "association":
				association, err := p.parseAssociation(decoder, token)
				if err != nil {
					return nil, err
				}
				coll.associationGroup = append(coll.associationGroup, association)
			case "collection":
				newColl, err := p.parseCollection(coll, decoder, token)
				if err != nil {
					return nil, err
				}
				coll.collectionGroup = append(coll.collectionGroup, newColl)
			}
		case xml.EndElement:
			coll.parent = parent
			if token.Name.Local == "collection" {
				if err = coll.init(); err != nil {
					return nil, err
				}
				return coll, nil
			}
		}
	}
	return coll, nil
}

func NewXMLConfigurationWithReader(fs fs.FS, reader io.Reader) (*Configuration, error) {
	parser := &XMLParser{FS: fs}
	return parser.Parse(reader)
}

func NewXMLConfiguration(filename string) (*Configuration, error) {
	return NewXMLConfigurationWithFS(LocalFS{}, filename)
}

// NewXMLConfigurationWithFS creates a new Configuration from an XML file.
func NewXMLConfigurationWithFS(fs fs.FS, filename string) (*Configuration, error) {
	baseDir := filepath.Dir(filename)
	fs = fsWrapper{fs, baseDir}
	filename = filepath.Base(filename)
	file, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return NewXMLConfigurationWithReader(fs, file)
}
