package juice

import (
	"regexp"
	"strings"

	"github.com/eatmoreapple/juice/driver"
)

var formatRegexp = regexp.MustCompile(`\$\{ *?([a-zA-Z0-9_\.]+) *?\}`)

// Statement defines a sql statement.
type Statement struct {
	mapper *Mapper
	action Action
	Nodes  []Node
	attrs  map[string]string
	name   string
	id     string
}

func (s *Statement) Attribute(key string) string {
	value := s.attrs[key]
	if value == "" {
		value = s.mapper.Attribute(key)
	}
	return value
}

func (s *Statement) setAttribute(key, value string) {
	if s.attrs == nil {
		s.attrs = make(map[string]string)
	}
	s.attrs[key] = value
}

func (s *Statement) ID() string {
	return s.id
}

func (s *Statement) Namespace() string {
	return s.mapper.Namespace()
}

func (s *Statement) lazyName() string {
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

// Name is a unique key of the whole statement.
func (s *Statement) Name() string {
	if s.name == "" {
		s.name = s.lazyName()
	}
	return s.name
}

func (s *Statement) Action() Action {
	return s.action
}

func (s *Statement) Accept(translator driver.Translator, p Parameter) (query string, args []any, err error) {
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

// Mapper is an getter of statements.
func (s *Statement) Mapper() *Mapper {
	return s.mapper
}

// Configuration returns the configuration of the statement.
func (s *Statement) Configuration() *Configuration {
	return s.mapper.Configuration()
}

// Engine returns the engine of the statement.
func (s *Statement) Engine() *Engine {
	return s.Configuration().engine
}

// ForRead returns true if the statement's Action is Select
func (s *Statement) ForRead() bool {
	return s.Action() == Select
}

// ForWrite returns true if the statement's Action is Insert, Update or Delete
func (s *Statement) ForWrite() bool {
	return s.Action() == Insert || s.Action() == Update || s.Action() == Delete
}

// IsInsert returns true if the statement's Action is Insert
func (s *Statement) IsInsert() bool {
	return s.Action() == Insert
}

// ResultMap returns the ResultMap of the statement.
func (s *Statement) ResultMap() (ResultMap, error) {
	key := s.Attribute("resultMap")
	if key == "" {
		return nil, ErrResultMapNotSet
	}
	return s.Mapper().GetResultMapByID(key)
}

// Build builds the statement with the given parameter.
func (s *Statement) Build(param Param) (query string, args []any, err error) {
	value := newGenericParam(param, s.Attribute("paramName"))

	translator := s.Engine().Driver().Translator()

	query, args, err = s.Accept(translator, value)
	if err != nil {
		return "", nil, err
	}
	if len(query) == 0 {
		return "", nil, ErrEmptyQuery
	}
	return query, args, nil
}

// QueryHandler returns the QueryHandler of the statement.
func (s *Statement) QueryHandler() QueryHandler {
	next := sessionQueryHandler()
	return s.Engine().middlewares.QueryContext(s, next)
}

// ExecHandler returns the ExecHandler of the statement.
func (s *Statement) ExecHandler() ExecHandler {
	next := sessionExecHandler()
	return s.Engine().middlewares.ExecContext(s, next)
}
