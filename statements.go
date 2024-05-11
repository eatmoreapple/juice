package juice

import (
	"regexp"
	"strings"

	"github.com/eatmoreapple/juice/driver"
)

type Statement interface {
	ID() string
	Name() string
	Attribute(key string) string
	Action() Action
	Configuration() IConfiguration
	ResultMap() (ResultMap, error)
	Build(translator driver.Translator, param Param) (query string, args []any, err error)
}

var formatRegexp = regexp.MustCompile(`\$\{ *?([a-zA-Z0-9_\.]+) *?\}`)

// xmlSQLStatement defines a sql xmlSQLStatement.
type xmlSQLStatement struct {
	mapper *Mapper
	action Action
	Nodes  []Node
	attrs  map[string]string
	name   string
	id     string
}

func (s *xmlSQLStatement) Attribute(key string) string {
	value := s.attrs[key]
	if value == "" {
		value = s.mapper.Attribute(key)
	}
	return value
}

func (s *xmlSQLStatement) setAttribute(key, value string) {
	if s.attrs == nil {
		s.attrs = make(map[string]string)
	}
	s.attrs[key] = value
}

func (s *xmlSQLStatement) ID() string {
	return s.id
}

func (s *xmlSQLStatement) Namespace() string {
	return s.mapper.Namespace()
}

func (s *xmlSQLStatement) lazyName() string {
	var builder = getBuilder()
	defer putBuilder(builder)
	if prefix := s.mapper.mappers.Prefix(); prefix != "" {
		builder.WriteString(prefix)
		builder.WriteString(".")
	}
	builder.WriteString(s.mapper.namespace)
	builder.WriteString(".")
	builder.WriteString(s.id)
	return builder.String()
}

// Name is a unique key of the whole xmlSQLStatement.
func (s *xmlSQLStatement) Name() string {
	if s.name == "" {
		s.name = s.lazyName()
	}
	return s.name
}

func (s *xmlSQLStatement) Action() Action {
	return s.action
}

func (s *xmlSQLStatement) Accept(translator driver.Translator, p Parameter) (query string, args []any, err error) {
	var builder = getBuilder()
	defer putBuilder(builder)
	for i, node := range s.Nodes {
		q, a, err := node.Accept(translator, p)
		if err != nil {
			return "", nil, err
		}
		if len(q) > 0 {
			builder.WriteString(q)
		}
		if len(a) > 0 {
			args = append(args, a...)
		}
		if i < len(s.Nodes)-1 && !strings.HasSuffix(q, " ") {
			builder.WriteString(" ")
		}
	}
	// format query
	// replace ${xxx} to an argument
	query = builder.String()
	return
}

// Configuration returns the configuration of the xmlSQLStatement.
func (s *xmlSQLStatement) Configuration() IConfiguration {
	return s.mapper.Configuration()
}

// ResultMap returns the ResultMap of the xmlSQLStatement.
func (s *xmlSQLStatement) ResultMap() (ResultMap, error) {
	key := s.Attribute("resultMap")
	if key == "" {
		return nil, ErrResultMapNotSet
	}
	return s.mapper.GetResultMapByID(key)
}

// Build builds the xmlSQLStatement with the given parameter.
func (s *xmlSQLStatement) Build(translator driver.Translator, param Param) (query string, args []any, err error) {
	value := newGenericParam(param, s.Attribute("paramName"))
	query, args, err = s.Accept(translator, value)
	if err != nil {
		return "", nil, err
	}
	if len(query) == 0 {
		return "", nil, ErrEmptyQuery
	}
	return query, args, nil
}
