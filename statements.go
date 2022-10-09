package juice

import (
	"strings"

	"github.com/eatmoreapple/juice/driver"
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

func (s *SampleStatement) Accept(translator driver.Translator, p Param) (query string, args []interface{}, err error) {
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

type StatementExecutor interface {
	Statement(v interface{}) Executor
}

type GenericStatementExecutor[result, param any] interface {
	Statement(v interface{}) GenericExecutor[result, param]
}

func NewGenericGenericStatementExecutor[result, param any](statementExecutor StatementExecutor) GenericStatementExecutor[result, param] {
	return &genericStatementExecutor[result, param]{statementExecutor}
}

type genericStatementExecutor[result any, param any] struct {
	statementExecutor StatementExecutor
}

func (s *genericStatementExecutor[result, param]) Statement(v any) GenericExecutor[result, param] {
	exe := s.statementExecutor.Statement(v)
	return &genericExecutor[result, param]{Executor: exe}
}
