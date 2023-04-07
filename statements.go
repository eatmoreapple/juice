package juice

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/eatmoreapple/juice/driver"
)

var formatRegexp = regexp.MustCompile(`\$\{([a-zA-Z0-9_\.]+)\}`)

// Statement defines a sql statement.
type Statement struct {
	engine *Engine
	mapper *Mapper
	action Action
	Nodes  []Node
	attrs  map[string]string
	name   string
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
	return s.Attribute("id")
}

func (s *Statement) Namespace() string {
	return s.mapper.Namespace()
}

// Name is a unique key of the whole statement.
func (s *Statement) Name() string {
	return s.name
}

func (s *Statement) Action() Action {
	return s.action
}

func (s *Statement) Accept(translator driver.Translator, p Param) (query string, args []interface{}, err error) {
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

	// cause the query may be a sql template, so we need to format it
	// for example, the query is "select * from ${table} where id = 1"
	// we need to replace ${table} to an argument
	query = formatRegexp.ReplaceAllStringFunc(query, func(find string) string {
		if err != nil {
			return find
		}
		param := formatRegexp.FindStringSubmatch(find)[1]
		value, exists := p.Get(param)
		if exists {
			return reflectValueToString(value)
		}
		// try to one from current statement attributes
		if attribute := s.Attribute(param); attribute == "" {
			err = fmt.Errorf("param %s not found in param or statement attributes", param)
			return find
		} else {
			return attribute
		}
	})
	return
}

func (s *Statement) String() string {
	return s.Name()
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
	return s.engine
}

// ForRead returns true if the statement's Action is Select
func (s *Statement) ForRead() bool {
	return s.Action() == Select
}

// ForWrite returns true if the statement's Action is Insert, Update or Delete
func (s *Statement) ForWrite() bool {
	return s.Action() == Insert || s.Action() == Update || s.Action() == Delete
}

// ResultMap returns the ResultMap of the statement.
func (s *Statement) ResultMap() (ResultMap, error) {
	key := s.Attribute("resultMap")
	if key == "" {
		return nil, ErrResultMapNotSet
	}
	return s.Mapper().GetResultMapByID(key)
}
