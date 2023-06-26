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
	ignoreEnv     bool
}

func (p XMLParser) IgnoreEnv(i bool) {
	p.ignoreEnv = i
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
				if p.ignoreEnv {
					continue
				}
				envs, err := p.parseEnvironments(decoder, token)
				if err != nil {
					return nil, err
				}
				p.configuration.Environments = *envs
			case "mappers":
				var mappers = Mappers{cfg: &p.configuration}
				if err := p.parseMappers(&mappers, token, decoder); err != nil {
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
		envs.SetAttr(attr.Name.Local, attr.Value)
	}
	if envs.Attr("default") == "" {
		return nil, errors.New("default environment is not specified")
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
	return nil, &nodeUnclosedError{nodeName: "environments"}
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
	return nil, &nodeUnclosedError{nodeName: "environment"}
}

func (p XMLParser) parseMappers(mappers *Mappers, start xml.StartElement, decoder *xml.Decoder) error {
	for _, attr := range start.Attr {
		mappers.setAttribute(attr.Name.Local, attr.Value)
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
			// mappers only support mapper child node
			if token.Name.Local == "mapper" {
				mapper, err := p.parseMapper(decoder, token)
				if err != nil {
					return err
				}
				// check result map
				if err = mapper.checkResultMap(); err != nil {
					return err
				}
				mapper.mappers = mappers
				for _, stmt := range mapper.statements {
					key := fmt.Sprintf("%s.%s", mapper.name(), stmt.ID())
					if err = mappers.setStatementByID(key, stmt); err != nil {
						return err
					}
				}
				// release memory
				mapper.statements = nil
			}
		case xml.EndElement:
			if token.Name.Local == "mappers" {
				return nil
			}
		}
	}
	return &nodeUnclosedError{nodeName: "mappers"}
}

func (p XMLParser) parseMapper(decoder *xml.Decoder, token xml.StartElement) (*Mapper, error) {
	mapper := &Mapper{}
	for _, attr := range token.Attr {
		mapper.setAttribute(attr.Name.Local, attr.Value)
	}

	resource := mapper.Attribute("resource")
	url := mapper.Attribute("url")
	namespace := mapper.Attribute("namespace")

	// check conflict
	// resource, url, namespace only one can be set
	// namespace is required if resource and url are not set
	switch {
	case resource != "" && url != "":
		return nil, &nodeAttributeConflictError{nodeName: "mapper", attrName: "resource|url"}
	case resource != "" && namespace != "":
		return nil, &nodeAttributeConflictError{nodeName: "mapper", attrName: "resource|namespace"}
	case url != "" && namespace != "":
		return nil, &nodeAttributeConflictError{nodeName: "mapper", attrName: "url|namespace"}
	case resource == "" && url == "" && namespace == "":
		return nil, &nodeAttributeRequiredError{nodeName: "mapper", attrName: "resource|url|namespace"}
	}
	if resource != "" {
		return p.parseMapperByResource(resource)
	}
	if url != "" {
		return p.parseMapperByURL(url)
	}
	if namespace == "" {
		return nil, &nodeAttributeRequiredError{nodeName: "mapper", attrName: "namespace"}
	}

	mapper.namespace = namespace
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
				key := stmt.ID()
				if _, exists := mapper.statements[key]; exists {
					return nil, fmt.Errorf("duplicate statement id: %s", key)
				}
				mapper.statements[key] = stmt
			case "sql":
				// parse sql node
				sqlNode := &SQLNode{mapper: mapper}
				if err = p.parseSQLNode(sqlNode, decoder, token); err != nil {
					return nil, err
				}
				if err = mapper.setSqlNode(sqlNode); err != nil {
					return nil, err
				}
			case "resultMap":
				// parse result map node
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
			switch token.Name.Local {
			case "values":
				if stmt.action != Insert {
					return fmt.Errorf("values node only support insert statement")
				}
				node, err := p.parseValuesNode(decoder)
				if err != nil {
					return err
				}
				stmt.Nodes = append(stmt.Nodes, node)
			case "alias":
				if stmt.action != Select {
					return fmt.Errorf("alias node only support select statement")
				}
				node, err := p.parseAliasNode(decoder)
				if err != nil {
					return err
				}
				stmt.Nodes = append(stmt.Nodes, node)
			default:
				node, err := p.parseTags(stmt.Mapper(), decoder, token)
				if err != nil {
					return err
				}
				stmt.Nodes = append(stmt.Nodes, node)
			}
		case xml.CharData:
			text := string(token)
			if char := strings.TrimSpace(text); char != "" {
				node, err := NewTextNode(char)
				if err != nil {
					return err
				}
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
		return nil, &nodeAttributeRequiredError{nodeName: "include", attrName: "refid"}
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
	return nil, &nodeUnclosedError{nodeName: "include"}
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
				node, err := NewTextNode(char)
				if err != nil {
					return nil, err
				}
				setNode.Nodes = append(setNode.Nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "set" {
				return setNode, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "set"}
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
		return nil, &nodeAttributeRequiredError{nodeName: "if", attrName: "test"}
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
				node, err := NewTextNode(char)
				if err != nil {
					return nil, err
				}
				ifNode.Nodes = append(ifNode.Nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "if" {
				return ifNode, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "if"}
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
				node, err := NewTextNode(char)
				if err != nil {
					return nil, err
				}
				whereNode.Nodes = append(whereNode.Nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "where" {
				return whereNode, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "where"}
}

func (p XMLParser) parseTrim(mapper *Mapper, decoder *xml.Decoder, token xml.StartElement) (Node, error) {
	trimNode := &TrimNode{}
	for _, attr := range token.Attr {
		switch attr.Name.Local {
		case "prefix":
			trimNode.Prefix = attr.Value
		case "prefixOverrides":
			prefixOverrides := strings.Split(attr.Value, "|")
			for i := range prefixOverrides {
				prefixOverrides[i] = strings.TrimSpace(prefixOverrides[i])
			}
			trimNode.PrefixOverrides = prefixOverrides
		case "suffix":
			trimNode.Suffix = attr.Value
		case "suffixOverrides":
			suffixOverrides := strings.Split(attr.Value, "|")
			for i := range suffixOverrides {
				suffixOverrides[i] = strings.TrimSpace(suffixOverrides[i])
			}
			trimNode.SuffixOverrides = suffixOverrides
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
	return nil, &nodeUnclosedError{nodeName: "trim"}
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

	// if collection is empty, use default param key instead.
	if foreachNode.Collection == "" {
		foreachNode.Collection = defaultParamKey
	}
	if foreachNode.Item == "" {
		return nil, &nodeAttributeRequiredError{nodeName: "foreach", attrName: "item"}
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
				node, err := NewTextNode(char)
				if err != nil {
					return nil, err
				}
				foreachNode.Nodes = append(foreachNode.Nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "foreach" {
				return foreachNode, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "foreach"}
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
	return nil, &nodeUnclosedError{nodeName: "choose"}
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
	return "", &nodeUnclosedError{nodeName: endElementName}
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
		return &nodeAttributeRequiredError{nodeName: "sql", attrName: "id"}
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
				node, err := NewTextNode(char)
				if err != nil {
					return err
				}
				sqlNode.nodes = append(sqlNode.nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "sql" {
				return nil
			}
		}
	}
	return &nodeUnclosedError{nodeName: "sql"}
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
		return nil, &nodeAttributeRequiredError{nodeName: "when", attrName: "test"}
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
				node, err := NewTextNode(char)
				if err != nil {
					return nil, err
				}
				whenNode.Nodes = append(whenNode.Nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "when" {
				return whenNode, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "when"}
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
				node, err := NewTextNode(char)
				if err != nil {
					return nil, err
				}
				otherwiseNode.Nodes = append(otherwiseNode.Nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "otherwise" {
				return otherwiseNode, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "otherwise"}
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
		return nil, &nodeAttributeRequiredError{nodeName: "resultMap", attrName: "id"}
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
				pk, err := p.parseResult(token, decoder, "id")
				if err != nil {
					return nil, err
				}
				resultMap.pk = pk
			case "result":
				result, err := p.parseResult(token, decoder, "result")
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
	return nil, &nodeUnclosedError{nodeName: "resultMap"}
}

func (p XMLParser) parseResult(token xml.StartElement, decoder *xml.Decoder, endTag string) (*resultNode, error) {
	result := &resultNode{}
	for _, attr := range token.Attr {
		switch attr.Name.Local {
		case "column":
			result.column = attr.Value
		case "property":
			result.property = attr.Value
		}
	}
	if result.column == "" {
		return nil, &nodeAttributeRequiredError{nodeName: endTag, attrName: "column"}
	}
	if result.property == "" {
		return nil, &nodeAttributeRequiredError{nodeName: endTag, attrName: "property"}
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch tp := token.(type) {
		case xml.EndElement:
			if tp.Name.Local == endTag {
				return result, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: endTag}
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
		return nil, &nodeAttributeRequiredError{nodeName: "association", attrName: "property"}
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
				result, err := p.parseResult(token, decoder, "result")
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
	return nil, &nodeUnclosedError{nodeName: "association"}
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
		return nil, &nodeAttributeRequiredError{nodeName: "collection", attrName: "property"}
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
				result, err := p.parseResult(token, decoder, "result")
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
	return nil, &nodeUnclosedError{nodeName: "collection"}
}

func (p XMLParser) parseValuesNode(decoder *xml.Decoder) (Node, error) {
	var node = make(ValuesNode, 0)
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil
		}
		switch token := token.(type) {
		case xml.StartElement:
			switch token.Name.Local {
			case "value":
				value, err := p.parseValueNode(token, decoder)
				if err != nil {
					return nil, err
				}
				node = append(node, value)
			}
		case xml.EndElement:
			if token.Name.Local == "values" {
				return node, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "values"}
}

func (p XMLParser) parseValueNode(token xml.StartElement, decoder *xml.Decoder) (*valueItem, error) {
	var ve valueItem
	for _, attr := range token.Attr {
		switch attr.Name.Local {
		case "value":
			ve.value = attr.Value
		case "column":
			ve.column = attr.Value
		}
	}
	if ve.column == "" {
		return nil, &nodeAttributeRequiredError{nodeName: "value", attrName: "column"}
	}
	if ve.value == "" {
		ve.value = fmt.Sprintf("#{%s}", ve.column)
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
		case xml.EndElement:
			if token.Name.Local == "value" {
				return &ve, nil
			}
		}
	}

	return nil, errors.New("value node requires value attribute to close")
}

// parseAliasNode parses the alias node
func (p XMLParser) parseAliasNode(decoder *xml.Decoder) (Node, error) {
	var node = make(SelectFieldAliasNode, 0)
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil
		}
		switch token := token.(type) {
		case xml.StartElement:
			switch token.Name.Local {
			case "field":
				item, err := p.parseFieldAlias(token, decoder)
				if err != nil {
					return nil, err
				}
				node = append(node, item)
			}
		case xml.EndElement:
			if token.Name.Local == "alias" {
				return node, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "alias"}
}

// parseFieldAlias parses the field alias node
func (p XMLParser) parseFieldAlias(token xml.StartElement, decoder *xml.Decoder) (*selectFieldAliasItem, error) {
	var item selectFieldAliasItem
	for _, attr := range token.Attr {
		switch attr.Name.Local {
		case "name":
			item.column = attr.Value
		case "alias":
			item.alias = attr.Value
		}
	}
	if item.column == "" {
		return nil, &nodeAttributeRequiredError{nodeName: "field", attrName: "name"}
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
		case xml.EndElement:
			if token.Name.Local == "field" {
				return &item, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "field"}
}

func NewXMLConfiguration(filename string) (*Configuration, error) {
	return NewXMLConfigurationWithFS(LocalFS{}, filename)
}

// NewXMLConfigurationWithFS creates a new Configuration from an XML file.
func NewXMLConfigurationWithFS(fs fs.FS, filename string) (*Configuration, error) {
	return newXMLConfigurationParser(fs, filename, false)
}

// newXMLConfigurationParser creates a new Configuration from an XML file which ignores environment parsing.
// for internal use only.
func newXMLConfigurationParser(fs fs.FS, filename string, ignoreEnv bool) (*Configuration, error) {
	baseDir := filepath.Dir(filename)
	fs = fsWrapper{fs, baseDir}
	filename = filepath.Base(filename)
	file, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	parser := &XMLParser{FS: fs, ignoreEnv: ignoreEnv}
	return parser.Parse(file)
}
