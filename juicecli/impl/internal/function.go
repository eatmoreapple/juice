package internal

import (
	"fmt"
	"strings"

	"github.com/eatmoreapple/juice"
)

type Function struct {
	// Name is a name of function.
	Name string
	// Args is an arguments of function.
	Args Values
	// Results is a results of function.
	Results Values
	// Receiver is a receiver of function.
	Receiver *Value
	// Body is a body of function.
	Body *string
	// Type is a type of function.
	Type string
	// Doc method document
	Doc *string
}

func (f Function) String() string {
	var builder strings.Builder
	if f.Doc != nil {
		builder.WriteString(*f.Doc)
	}
	builder.WriteString("func ")
	if f.Receiver != nil {
		builder.WriteString(fmt.Sprintf("(%s) ", f.Receiver))
	}
	builder.WriteString(f.Name)
	builder.WriteString(fmt.Sprintf("%s", f.Args))
	if len(f.Results) > 0 {
		builder.WriteString(fmt.Sprintf(" %s", f.Results))
	}
	builder.WriteString(" {")
	if f.Body != nil {
		builder.WriteString(*f.Body)
	} else {
		builder.WriteString("\n\tpanic(\"not implemented\")")
	}
	builder.WriteString("\n}")
	return formatCode(builder.String())
}

type Functions []*Function

type FunctionBodyMaker struct {
	statement *juice.Statement
	function  *Function
}

func (f *FunctionBodyMaker) Make() error {
	var bodyMaker functionBodyMaker
	if f.statement.ForRead() {
		bodyMaker = &readFuncBodyMaker{function: f.function, statement: f.statement}
	} else {
		bodyMaker = &writeFuncBodyMaker{function: f.function, statement: f.statement}
	}
	return bodyMaker.Make()
}

type functionBodyMaker interface {
	Make() error
}

type readFuncBodyMaker struct {
	statement *juice.Statement
	function  *Function
}

func (f *readFuncBodyMaker) Make() error {
	if err := f.check(); err != nil {
		return err
	}
	f.build()
	return nil
}

func (f *readFuncBodyMaker) check() error {
	if len(f.function.Results) != 2 {
		return fmt.Errorf("%s: must have two results", f.function.Name)
	}
	if f.function.Results[1].Type != "error" {
		return fmt.Errorf("%s: second result must be error", f.function.Name)
	}
	if len(f.function.Args) == 0 {
		return fmt.Errorf("%s: must have at least one argument", f.function.Name)
	}
	if f.function.Args[0].TypeName() != "context.Context" {
		return fmt.Errorf("%s: first argument must be context.Context", f.function.Name)
	}
	return nil
}

func (f *readFuncBodyMaker) build() {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("\n\tmanager := juice.ManagerFromContext(%s)", f.function.Args[0].Name))
	builder.WriteString(fmt.Sprintf("\n\tvar iface %s = %s", f.function.Type, f.function.Receiver.Name))
	builder.WriteString(fmt.Sprintf("\n\texecutor := juice.NewGenericManager[%s](manager).Object(iface.%s)",
		f.function.Results[0].TypeName(),
		f.function.Name))
	query := f.function.Args.AsQuery()
	builder.WriteString(fmt.Sprintf("\n\treturn executor.QueryContext(%s, %s)", f.function.Args[0].Name, query))
	body := formatCode(builder.String())
	f.function.Body = &body
}

type writeFuncBodyMaker struct {
	statement *juice.Statement
	function  *Function
}

func (f *writeFuncBodyMaker) Make() error {
	if err := f.check(); err != nil {
		return err
	}
	f.build()
	return nil
}

func (f *writeFuncBodyMaker) check() error {
	if len(f.function.Args) == 0 {
		return fmt.Errorf("%s: must have at least one argument", f.function.Name)
	}
	if f.function.Args[0].TypeName() != "context.Context" {
		return fmt.Errorf("%s: first argument must be context.Context", f.function.Name)
	}
	if len(f.function.Results) == 0 {
		return fmt.Errorf("%s: must have one result", f.function.Name)
	}
	if len(f.function.Results) == 1 {
		if f.function.Results[0].Type != "error" {
			return fmt.Errorf("%s: result must be error", f.function.Name)
		}
	}
	if len(f.function.Results) == 2 {
		if f.function.Results[0].TypeName() != "sql.Result" {
			return fmt.Errorf("%s: first result must be sql.Result", f.function.Name)
		}
		if f.function.Results[1].Type != "error" {
			return fmt.Errorf("%s: second result must be error", f.function.Name)
		}
	}
	if len(f.function.Results) > 2 {
		return fmt.Errorf("%s: must have at most two results", f.function.Name)
	}
	return nil
}

func (f *writeFuncBodyMaker) build() {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("\n\tmanager := juice.ManagerFromContext(%s)", f.function.Args[0].Name))
	builder.WriteString(fmt.Sprintf("\n\tvar iface %s = %s", f.function.Type, f.function.Receiver.Name))
	//key := fmt.Sprintf(`"%s.%s"`, f.function.Namespace, f.function.name)
	builder.WriteString(fmt.Sprintf("\n\texecutor := manager.Object(iface.%s)", f.function.Name))
	query := f.function.Args.AsQuery()
	if len(f.function.Results) == 1 {
		builder.WriteString(fmt.Sprintf("\n\t_, err := executor.ExecContext(%s, %s)", f.function.Args[0].Name, query))
		builder.WriteString("\n\treturn err")
	} else {
		builder.WriteString(fmt.Sprintf("\n\treturn executor.ExecContext(%s, %s)", f.function.Args[0].Name, query))
	}
	body := formatCode(builder.String())
	f.function.Body = &body
}
