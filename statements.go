package juice

import (
	"strings"

	"github.com/eatmoreapple/juice/driver"
)

// Statement defines the interface for a statement
type Statement interface {
	Node

	// ID returns the id of the statement
	ID() string

	// Namespace returns the namespace of the statement
	Namespace() string

	// Action returns the action of the statement
	Action() Action
}

// SampleStatement implements the Statement interface
type SampleStatement struct {
	id        string
	namespace string
	action    Action
	Nodes     []Node
}

func (s *SampleStatement) ID() string {
	return s.id
}

func (s *SampleStatement) Namespace() string {
	return s.namespace
}

func (s *SampleStatement) Action() Action {
	return s.action
}

func (s *SampleStatement) Accept(translator driver.Translator, p Param) (query string, args []interface{}, err error) {
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

// statementIdentity returns the identity of the statement
func statementIdentity(statement Statement) string {
	return statement.Namespace() + "." + statement.ID()
}
