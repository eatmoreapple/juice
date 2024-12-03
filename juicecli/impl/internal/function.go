/*
Copyright 2023 eatmoreapple

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package internal

import (
	"errors"
	"fmt"
	stdast "go/ast"
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
	statement juice.Statement
	function  *Function
}

func (f *FunctionBodyMaker) Make() error {
	var bodyMaker functionBodyMaker
	if f.statement.Action().ForRead() {
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
	statement juice.Statement
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

	var body string

	retType := f.function.Results()[0].TypeName()
	query := formatParams(f.function.Params())

	isArrayType := strings.HasPrefix(retType, "[]")

	_, err := f.statement.ResultMap()

	// if isArrayType is true and the error is ErrResultMapNotSet
	if isArrayType && errors.Is(err, juice.ErrResultMapNotSet) {
		// if is an array type
		retType = retType[2:]
		isPointer := strings.HasPrefix(retType, "*")
		if isPointer {
			retType = retType[1:]
		}
		fmt.Fprintf(builder, "\n\trows, err := manager.Object(iface.%s).QueryContext(%s, %s)",
			f.function.Name(), f.function.Params().NameAt(ast.ParamPrefix, 0), query)
		fmt.Fprintf(builder, "\n\tif err != nil {")
		fmt.Fprintf(builder, "\n\t\treturn nil, err")
		fmt.Fprintf(builder, "\n\t}")
		fmt.Fprintf(builder, "\n\tdefer func() { _ = rows.Close() }()")
		if !isPointer {
			fmt.Fprintf(builder, "\n\treturn juice.List[%s](rows)", retType)
		} else {
			fmt.Fprintf(builder, "\n\tret, err := juice.List[%s](rows)", retType)
			fmt.Fprintf(builder, "\n\tvar result = make([]*%s, len(ret))", retType)
			fmt.Fprintf(builder, "\n\tfor index, item := range ret {")
			fmt.Fprintf(builder, "\n\t\tresult[index] = &item")
			fmt.Fprintf(builder, "\n\t}")
			fmt.Fprintf(builder, "\n\treturn result, err")
		}
		body = formatCode(builder.String())
	} else {
		// if is a pointer
		isPointer := strings.HasPrefix(retType, "*")
		if isPointer {
			// if is a pointer, remove the *
			// in order to get the real type and use it to create the object without using reflection.
			retType = retType[1:]
		}
		fmt.Fprintf(builder, "\n\texecutor := juice.NewGenericManager[%s](manager).Object(iface.%s)", retType, f.function.Name())
		fmt.Fprintf(builder, "\n\tret, err := executor.QueryContext(%s, %s)", f.function.Params().NameAt(ast.ParamPrefix, 0), query)
		if isPointer {
			fmt.Fprintf(builder, "\n\treturn &ret, err")
		} else {
			fmt.Fprintf(builder, "\n\treturn ret, err")
		}
		body = formatCode(builder.String())
	}

	f.function.body = body
}

type writeFuncBodyMaker struct {
	statement juice.Statement
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
	// check input params
	params := f.function.Params()

	switch len(params) {
	case 0:
		return fmt.Errorf("%s: must have at least one argument", f.function.Name())
	case 1:
		if params[0].TypeName() != "context.Context" {
			return fmt.Errorf("%s: first argument must be context.Context", f.function.Name())
		}
	case 2:
		if params[0].TypeName() != "context.Context" {
			return fmt.Errorf("%s: first argument must be context.Context", f.function.Name())
		}
		// if `useGeneratedKeys` is true, the second parameter must be a pointer or a pointer array type
		useGeneratedKeys := f.statement.Attribute("useGeneratedKeys")
		if useGeneratedKeys == "true" {
			// if the second parameter is not a pointer
			param1 := params[1]
			if arrayType, ok := param1.Field.Type.(*stdast.ArrayType); ok {
				// if arrayType.Elt is not a pointer
				starType, ok := arrayType.Elt.(*stdast.StarExpr)
				if !ok {
					return fmt.Errorf("`%s` `useGeneratedKeys` is true, but `%s` is not a pointer array type", f.statement.ID(), param1.Name())
				}
				// todo check the starType.X is a struct type
				_ = starType
			} else {
				// not an array type
				// ensure it is a pointer struct type
				starType, ok := param1.Field.Type.(*stdast.StarExpr)
				if !ok {
					return fmt.Errorf("`%s` `useGeneratedKeys` is true, but `%s` is not a pointer type", f.statement.ID(), param1.Name())
				}
				// todo check the starType.X is a struct type
				_ = starType
			}
		}
	default:
		// more than 2 parameters
		// if `useGeneratedKeys` is true
		useGeneratedKeys := f.statement.Attribute("useGeneratedKeys")
		if useGeneratedKeys == "true" {
			return fmt.Errorf("`%s` `useGeneratedKeys` is true, but there are more than 2 parameters", f.statement.ID())
		}
	}

	// check results

	results := f.function.Results()

	switch len(results) {
	case 0:
		return fmt.Errorf("%s: must have one result", f.function.Name())
	case 1:
		if results[0].TypeName() != "error" {
			return fmt.Errorf("%s: result must be error", f.function.Name())
		}
	case 2:
		if results[0].TypeName() != "sql.Result" {
			return fmt.Errorf("%s: first result must be sql.Result", f.function.Name())
		}
		if results[1].TypeName() != "error" {
			return fmt.Errorf("%s: second result must be error", f.function.Name())
		}
	default:
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
		param1 := params[1]
		if param1.IsBuiltInType() {
			return fmt.Sprintf(`juice.H{"%s": %s}`, param1.Name(), param1.Name())
		}
		switch param1.Field.Type.(type) {
		case *stdast.ArrayType:
			return fmt.Sprintf(`juice.H{"%s": %s}`, param1.Name(), param1.Name())
		}
		return param1.Name()
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
