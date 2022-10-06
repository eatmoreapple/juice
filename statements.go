package pillow

import (
	"github.com/eatmoreapple/pillow/driver"
	"strings"
)

type Statement interface {
	Node
	ID() string
	Namespace() string
}

type SampleStatement struct {
	id        string
	namespace string
	Nodes     []Node
}

func (s *SampleStatement) ID() string {
	return s.id
}

func (s *SampleStatement) Namespace() string {
	return s.namespace
}

func (s *SampleStatement) Accept(translator driver.Translate, p Param) (query string, args []interface{}, err error) {
	var builder strings.Builder
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
