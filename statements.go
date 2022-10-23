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
	mapper *Mapper
	action Action
	Nodes  []Node
	attrs  map[string]string
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

// Key is a unique key of the whole statement.
func (s *Statement) Key() string {
	return s.Namespace() + "." + s.ID()
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
	if len(p) > 0 || len(s.attrs) > 1 {
		query = formatRegexp.ReplaceAllStringFunc(query, func(find string) string {
			if err != nil {
				return find
			}
			param := formatRegexp.FindStringSubmatch(find)[1]
			value, exists := p.Get(param)
			if exists {
				return reflectValueToString(value)
			}
			// try to get from current statement attributes
			if attribute := s.Attribute(param); attribute == "" {
				err = fmt.Errorf("param %s not found in param or statement attributes", param)
				return find
			} else {
				return attribute
			}
		})
	}
	return query, args, nil
}

func (s *Statement) String() string {
	return s.Key()
}

// Mapper is an getter of statements.
func (s *Statement) Mapper() *Mapper {
	return s.mapper
}
