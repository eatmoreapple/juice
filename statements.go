package juice

import (
	"strings"

	"github.com/eatmoreapple/juice/driver"
)

// Statement defines a sql statement.
type Statement struct {
	mapper *Mapper
	action Action
	Nodes  []Node
	attrs  map[string]string
}

func (s *Statement) Attribute(key string) string {
	return s.attrs[key]
}

func (s *Statement) SetAttribute(key, value string) {
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
		builder.WriteString(q)
		args = append(args, a...)
		if i < len(s.Nodes)-1 && !strings.HasSuffix(q, " ") {
			builder.WriteString(" ")
		}
	}
	return builder.String(), args, nil
}
