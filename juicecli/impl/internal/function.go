package internal

import (
	"fmt"
	"strings"

	"github.com/eatmoreapple/juice"
	"github.com/eatmoreapple/juice/juicecli/internal/ast"
)

type Function struct {
	method   *ast.Function
	receiver string
	body     string
	typename string
}

func (f *Function) String() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("func (%s %s) %s", f.receiverAlias(), f.receiver, f.method.Signature()))
	builder.WriteString(" {")
	if f.body == "" {
		builder.WriteString("panic(\"not implemented\")")
	} else {
		builder.WriteString(f.body)
	}
	builder.WriteString("\n")
	builder.WriteString("}")
	return builder.String()
}

func (f *Function) receiverAlias() string {
	return strings.ToLower(f.receiver[:1])
}

func (f *Function) Params() ast.ValueGroup {
	return f.method.Params()
}

func (f *Function) Results() ast.ValueGroup {
	return f.method.Results()
}

func (f *Function) Name() string {
	return f.method.Name()
}

type FunctionGroup []*Function

func (f FunctionGroup) String() string {
	var builder strings.Builder
	for index, function := range f {
		builder.WriteString(function.String())
		if index < len(f)-1 {
			builder.WriteString("\n\n")
		}
	}
	return builder.String()
}

type FunctionBodyMaker struct {
	statement *juice.Statement
	function  *Function
	typename  string
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
	if len(f.function.method.Results()) != 2 {
		return fmt.Errorf("%s: must have two results", f.function.method.Name())
	}
	if f.function.Results()[1].TypeName() != "error" {
		return fmt.Errorf("%s: second result must be error", f.function.method.Names)
	}
	if len(f.function.Params()) == 0 {
		return fmt.Errorf("%s: must have at least one argument", f.function.Name())
	}
	if f.function.Params()[0].TypeName() != "context.Context" {
		return fmt.Errorf("%s: first argument must be context.Context", f.function.Name())
	}
	return nil
}

func (f *readFuncBodyMaker) build() {
	var builder = new(strings.Builder)
	fmt.Fprintf(builder, "\n\tmanager := juice.ManagerFromContext(%s)", f.function.Params().NameAt(ast.ParamPrefix, 0))
	fmt.Fprintf(builder, "\n\tvar iface %s = %s", f.function.typename, f.function.receiverAlias())
	retType := f.function.Results()[0].TypeName()
	// if is a pointer
	isPointer := strings.HasPrefix(retType, "*")
	if isPointer {
		// if is a pointer, remove the *
		// in order to get the real type and use it to create the object without using reflection.
		retType = retType[1:]
	}
	fmt.Fprintf(builder, "\n\texecutor := juice.NewGenericManager[%s](manager).Object(iface.%s)", retType, f.function.Name())
	query := formatParams(f.function.Params())
	fmt.Fprintf(builder, "\n\tret, err := executor.QueryContext(%s, %s)", f.function.Params().NameAt(ast.ParamPrefix, 0), query)
	if isPointer {
		fmt.Fprintf(builder, "\n\treturn &ret, err")
	} else {
		fmt.Fprintf(builder, "\n\treturn ret, err")
	}
	body := formatCode(builder.String())
	f.function.body = body
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
	if len(f.function.Params()) == 0 {
		return fmt.Errorf("%s: must have at least one argument", f.function.Name())
	}
	if f.function.Params()[0].TypeName() != "context.Context" {
		return fmt.Errorf("%s: first argument must be context.Context", f.function.Name())
	}
	if len(f.function.Results()) == 0 {
		return fmt.Errorf("%s: must have one result", f.function.Name())
	}
	if len(f.function.Results()) == 1 {
		if f.function.Results()[0].TypeName() != "error" {
			return fmt.Errorf("%s: result must be error", f.function.Name())
		}
	}
	if len(f.function.Results()) == 2 {
		if f.function.Results()[0].TypeName() != "sql.Result" {
			return fmt.Errorf("%s: first result must be sql.Result", f.function.Name())
		}
		if f.function.Results()[1].TypeName() != "error" {
			return fmt.Errorf("%s: second result must be error", f.function.Name())
		}
	}
	if len(f.function.Results()) > 2 {
		return fmt.Errorf("%s: must have at most two results", f.function.Name())
	}
	return nil
}

func (f *writeFuncBodyMaker) build() {
	var builder = new(strings.Builder)
	fmt.Fprintf(builder, "\n\tmanager := juice.ManagerFromContext(%s)", f.function.Params().NameAt(ast.ParamPrefix, 0))
	fmt.Fprintf(builder, "\n\tvar iface %s = %s", f.function.typename, f.function.receiverAlias())
	fmt.Fprintf(builder, "\n\texecutor := juice.NewGenericManager[any](manager).Object(iface.%s)", f.function.Name())
	query := formatParams(f.function.Params())
	if len(f.function.Results()) == 1 {
		fmt.Fprintf(builder, "\n\t_, err := executor.ExecContext(%s, %s)", f.function.Params()[0].Name(), query)
		fmt.Fprintf(builder, "\n\treturn err")
	} else {
		fmt.Fprintf(builder, "\n\treturn executor.ExecContext(%s, %s)", f.function.Params()[0].Name(), query)
	}
	body := formatCode(builder.String())
	f.function.body = body
}

func formatParams(params ast.ValueGroup) string {
	switch len(params) {
	case 0, 1:
		return "nil"
	case 2:
		return params[1].Name()
	default:
		var builder strings.Builder
		builder.WriteString("juice.H{")
		for index, param := range params[1:] {
			builder.WriteString(fmt.Sprintf("%q: %s", param.Name(), param.Name()))
			if index < len(params)-2 {
				builder.WriteString(", ")
			}
		}
		builder.WriteString("}")
		return builder.String()
	}
}
