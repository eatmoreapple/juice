package juice

import (
	"strings"

	"github.com/eatmoreapple/juice/driver"
)

// Statement defines a sql statement.
type Statement struct {
	id        string
	namespace string
	action    Action
	Nodes     []Node
	paramName string
}

func (s *Statement) ID() string {
	return s.id
}

func (s *Statement) Namespace() string {
	return s.namespace
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

func (s *Statement) Key() string {
	return s.Namespace() + "." + s.ID()
}

func (s *Statement) ParamName() string {
	if s.paramName == "" {
		return defaultParamKey
	}
	return s.paramName
}
