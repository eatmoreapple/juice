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

package eval

import (
	"errors"
	"fmt"
	"github.com/eatmoreapple/juice/expr"
	"github.com/eatmoreapple/juice/internal/env"
	"github.com/eatmoreapple/juice/internal/reflectlite"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"strings"
)

// SyntaxError represents a syntax error.
// The error occurs when parsing the expression.
type SyntaxError struct {
	err error
}

// Error returns the error message.
func (s *SyntaxError) Error() string {
	return fmt.Sprintf("syntax error: %v", s.err)
}

// Unwrap returns the underlying error.
func (s *SyntaxError) Unwrap() error {
	return s.err
}

// ExprPretreatment is an expression pretreatment.
// It is used to pretreatment the expression before parsing.
type ExprPretreatment interface {
	PretreatmentExpr(expr string) (string, error)
}

// ExprPretreatmentChain is an expression pretreatment chain.
type ExprPretreatmentChain []ExprPretreatment

// PretreatmentExpr implements the ExprPretreatment interface.
func (e ExprPretreatmentChain) PretreatmentExpr(expr string) (string, error) {
	var err error
	for _, pretreatment := range e {
		expr, err = pretreatment.PretreatmentExpr(expr)
		if err != nil {
			return "", err
		}
	}
	return expr, nil
}

// exprKeyWordReplacePretreatment is an expression pretreatment that replaces the keyword.
type exprKeyWordReplacePretreatment struct {
	keyword string
	replace string
}

// PretreatmentExpr implements the ExprPretreatment interface.
func (e *exprKeyWordReplacePretreatment) PretreatmentExpr(expr string) (string, error) {
	return strings.Replace(expr, e.keyword, e.replace, -1), nil
}

var (
	// FIXME: use a better way to replace the keyword.

	// andReplacePretreatment is an expression pretreatment that replaces "and" with "&&".
	andReplacePretreatment ExprPretreatment = &exprKeyWordReplacePretreatment{
		keyword: " and ", // must have space
		replace: " && ",
	}
	// orReplacePretreatment is an expression pretreatment that replaces "or" with "||".
	orReplacePretreatment ExprPretreatment = &exprKeyWordReplacePretreatment{
		keyword: " or ", // must have space
		replace: " || ",
	}

	// exprPretreatmentChain is an expression pretreatment chain.
	exprPretreatmentChain ExprPretreatment = ExprPretreatmentChain{
		andReplacePretreatment,
		orReplacePretreatment,
	}
)

// ExprCompiler is an evaluator of the expression.
type ExprCompiler interface {
	// Compile compiles the expression and returns the expression.
	Compile(expr string) (Expression, error)
}

// Value is an alias of reflect.Value.
// for semantic.
type Value = reflect.Value

// Expression is an expression which can be evaluated to a value.
type Expression interface {
	// Execute evaluates the expression and returns the value.
	Execute(params Parameter) (Value, error)
}

// goExprCompiler is an evaluator of the expression who uses the go/ast package.
type goExprCompiler struct {
	pretreatment ExprPretreatment
}

// Compile compiles the expression and returns the expression.
func (e *goExprCompiler) Compile(expr string) (Expression, error) {
	// pretreatment the expression first.
	expr_, err := e.pretreatment.PretreatmentExpr(expr)
	if err != nil {
		return nil, err
	}
	// parse the expression with go/ast.
	exp, err := parser.ParseExpr(expr_)
	if err != nil {
		return nil, &SyntaxError{err}
	}
	return &goExpression{exp}, nil
}

// goExpression is an expression who uses the go/ast package.
type goExpression struct {
	ast.Expr
}

// Execute evaluates the expression and returns the value.
func (e *goExpression) Execute(params Parameter) (Value, error) {
	return eval(e.Expr, params)
}

var (
	// DefaultExprCompiler is the default evaluator.
	// Reset it to change the default behavior.
	DefaultExprCompiler ExprCompiler = &goExprCompiler{pretreatment: exprPretreatmentChain}
)

// Compile compiles the expression and returns the expression.
func Compile(expr string) (Expression, error) {
	return DefaultExprCompiler.Compile(expr)
}

// Eval is a shortcut of DefaultEvaluator.Compiler(expr).Execute(params).
func Eval(expr string, params Parameter) (Value, error) {
	// cache the compiler, incase the DefaultExprCompiler is changed by other goroutine.
	compiler := DefaultExprCompiler
	if compiler == nil {
		return reflect.Value{}, errors.New("evaluator is nil")
	}
	expression, err := compiler.Compile(expr)
	if err != nil {
		return reflect.Value{}, err
	}
	return expression.Execute(params)
}

func eval(exp ast.Expr, params Parameter) (reflect.Value, error) {
	switch exp := exp.(type) {
	case *ast.BinaryExpr:
		return evalBinaryExpr(exp, params)
	case *ast.ParenExpr:
		return eval(exp.X, params)
	case *ast.BasicLit:
		return evalBasicLit(exp)
	case *ast.Ident:
		return evalIdent(exp, params)
	case *ast.SelectorExpr:
		return evalSelectorExpr(exp, params)
	case *ast.CallExpr:
		return evalCallExpr(exp, params)
	case *ast.UnaryExpr:
		return evalUnaryExpr(exp, params)
	case *ast.IndexExpr:
		return evalIndexExpr(exp, params)
	case *ast.StarExpr:
		return eval(exp.X, params)
	case *ast.SliceExpr:
		return evalSliceExpr(exp, params)
	default:
		return reflect.Value{}, fmt.Errorf("unsupported expression: %T", exp)
	}
}

func evalSliceExpr(exp *ast.SliceExpr, params Parameter) (reflect.Value, error) {
	value, err := eval(exp.X, params)
	if err != nil {
		return reflect.Value{}, err
	}

	value = reflectlite.Unwrap(value)

	var low, high int

	// like [1:] expr
	// if exp.Low is nil, it means the slice starts from 0
	if exp.Low != nil {
		low, err = strconv.Atoi(exp.Low.(*ast.BasicLit).Value)
		if err != nil {
			return reflect.Value{}, err
		}
	}
	// like [:1] expr
	if exp.High != nil {
		high, err = strconv.Atoi(exp.High.(*ast.BasicLit).Value)
		if err != nil {
			return reflect.Value{}, err
		}
	} else {
		// otherwise, it means the slice ends at the end of the slice
		high = value.Len()
	}
	if !exp.Slice3 {
		return value.Slice(low, high), nil
	}
	// like [1:2:3] expr
	// if exp.Max is nil, it means the capacity of the slice
	var sliceMax int
	if exp.Max != nil {
		sliceMax, err = strconv.Atoi(exp.Max.(*ast.BasicLit).Value)
		if err != nil {
			return reflect.Value{}, err
		}
	}
	return value.Slice3(low, high, sliceMax), nil
}

func evalUnaryExpr(exp *ast.UnaryExpr, params Parameter) (reflect.Value, error) {
	value, err := eval(exp.X, params)
	if err != nil {
		return reflect.Value{}, err
	}
	switch exp.Op {
	case token.SUB:
		return reflect.ValueOf(-value.Int()), nil
	case token.ADD:
		return reflect.ValueOf(+value.Int()), nil
	case token.NOT:
		return reflect.ValueOf(!value.Bool()), nil
	case token.XOR:
		return reflect.ValueOf(^value.Int()), nil
	case token.AND:
		return reflect.ValueOf(^value.Int()), nil
	case token.MUL:
		return reflect.ValueOf(value.Pointer()), nil
	default:
		return reflect.Value{}, errors.New("unsupported unary expression")
	}
}

func evalIndexExpr(exp *ast.IndexExpr, params Parameter) (reflect.Value, error) {
	value, err := eval(exp.X, params)
	if err != nil {
		return reflect.Value{}, err
	}
	value = reflectlite.Unwrap(value)

	index, err := eval(exp.Index, params)
	if err != nil {
		return reflect.Value{}, err
	}
	switch value.Kind() {
	case reflect.Array, reflect.Slice, reflect.String:
		i := index.Int()
		if i >= int64(value.Len()) {
			return reflect.Value{}, errors.New("index out of range")
		}
		return value.Index(int(i)), nil
	case reflect.Map:
		// in this case, index must be assignable to the map's key type
		// if value not exist, return the map's default value
		v := value.MapIndex(index)
		if v.IsValid() {
			return v, nil
		}
		// get map default value
		if v.Kind() == reflect.Interface {
			v = v.Elem()
		}
		if v.Kind() == reflect.Invalid {
			v = reflect.Zero(value.Type().Elem())
		}
		return v, nil
	default:
		return reflect.Value{}, fmt.Errorf("invalid index expression: %v", value.Kind())
	}
}

func evalCallExpr(exp *ast.CallExpr, params Parameter) (reflect.Value, error) {
	fn, err := eval(exp.Fun, params)
	if err != nil {
		return reflect.Value{}, err
	}
	if fn.Kind() == reflect.Interface {
		fn = fn.Elem()
	}
	if fn.Kind() != reflect.Func {
		return reflect.Value{}, errors.New("unsupported call expression")
	}
	fnType := fn.Type()
	if numIn := fnType.NumIn(); numIn != len(exp.Args) {
		return reflect.Value{}, fmt.Errorf("invalid number of arguments: expected %d, got %d", numIn, len(exp.Args))
	}
	// TODO dot dot dot support
	// ...type
	if fnType.NumOut() != 2 {
		return reflect.Value{}, fmt.Errorf("invalid number of return values: expected 2, got %d", fn.Type().NumOut())
	}
	// evaluate the arguments
	args := make([]reflect.Value, 0, len(exp.Args))
	for i, arg := range exp.Args {
		value, err := eval(arg, params)
		if err != nil {
			return reflect.Value{}, err
		}
		value = reflectlite.Unwrap(value)
		// type conversion for function arguments
		in := fnType.In(i)
		if in.Kind() != value.Kind() {
			if !value.CanConvert(in) {
				return reflect.Value{}, fmt.Errorf("cannot convert %s to %s", value.Type().Name(), in.Name())
			}
			value = value.Convert(in)
		}
		args = append(args, value)
	}
	// call the function
	rets := fn.Call(args)
	// unreachable code.
	// just for nil check
	if len(rets) != 2 {
		return reflect.Value{}, errors.New("invalid number of return values")
	}
	// check if the function returns an error
	errRet := rets[1]
	if !errRet.IsNil() {
		// the second return value must be an error

		// we need to check if the second return value implements the error interface

		// try to convert the second return value to error
		if ok := errRet.Type().Implements(errType); ok {
			// I believe this is always true
			return reflect.Value{}, errRet.Interface().(error)
		}
		// this should never happen, but just in case
		// should I mark it unreachable?
		return reflect.Value{}, errors.New("cannot convert return value to error")
	}
	return rets[0], nil
}

func evalSelectorExpr(exp *ast.SelectorExpr, params Parameter) (reflect.Value, error) {
	if exp.Sel == nil {
		return reflect.Value{}, errors.New("invalid selector expression")
	}

	fieldOrTagOrMethodName := exp.Sel.Name

	if len(fieldOrTagOrMethodName) == 0 {
		return reflect.Value{}, errors.New("invalid selector expression")
	}

	x, err := eval(exp.X, params)
	if err != nil {
		return reflect.Value{}, err
	}

	unwarned := reflectlite.Unwrap(x)

	// check if the field name is exported
	isExported := token.IsExported(fieldOrTagOrMethodName)

	var result reflect.Value

	switch unwarned.Kind() {
	case reflect.Struct:
		// findFromTag is a closure function that tries to find the field from the field tag
		findFromTag := func() {
			find := reflectlite.From(unwarned).FindFieldFromTag(env.DefaultParamKey, fieldOrTagOrMethodName)
			if find.IsValid() {
				result = find.Value
			}
		}

		// unexported field cannot be accessed, so we try to find from the field tag
		if !isExported {
			// find from the field tag
			findFromTag()
		} else {
			// find from the field name first
			if unwarned.NumField() > 0 {
				result = unwarned.FieldByName(fieldOrTagOrMethodName)
			}

			// try to find from the field tag
			if !result.IsValid() {
				findFromTag()
			}
		}
	case reflect.Map:
		result = unwarned.MapIndex(reflect.ValueOf(fieldOrTagOrMethodName))
		// select expression does not support get default value from map
		// it might be ambiguous with calling a method
	}

	// try to find method from the type
	if isExported && x.NumMethod() > 0 {
		// use x directly, in case x is a pointer
		result = x.MethodByName(fieldOrTagOrMethodName)
	}

	// we failed to find the field
	// it means you wrote a wrong expression
	if !result.IsValid() {
		return reflect.Value{}, fmt.Errorf("invalid selector expression: %s", fieldOrTagOrMethodName)
	}

	return result, nil
}

func evalIdent(exp *ast.Ident, params Parameter) (reflect.Value, error) {
	if fn, ok := builtins[exp.Name]; ok {
		return fn, nil
	}
	value, ok := params.Get(exp.Name)
	if !ok {
		return reflect.Value{}, fmt.Errorf("undefined identifier: %s", exp.Name)
	}
	return value, nil
}

func evalBasicLit(exp *ast.BasicLit) (reflect.Value, error) {
	switch exp.Kind {
	case token.INT:
		value, err := strconv.ParseInt(exp.Value, 10, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(value), nil
	case token.FLOAT:
		value, err := strconv.ParseFloat(exp.Value, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(value), nil
	case token.STRING, token.CHAR:
		return reflect.ValueOf(exp.Value[1 : len(exp.Value)-1]), nil
	default:
		return reflect.Value{}, errors.New("unsupported basic literal")
	}
}

// evalFunc evaluates a function call expression.
func evalFunc(fn reflect.Value, exp *ast.BinaryExpr, params Parameter) reflect.Value {
	var args []reflect.Value
	if exp.Y != nil {
		arg, err := eval(exp.Y, params)
		if err != nil {
			return reflect.Value{}
		}
		args = append(args, arg)
	}
	out := fn.Call(args)
	if len(out) == 0 {
		return reflect.Value{}
	}
	return out[0]
}

// evalBinaryExpr evaluates a binary expression.
func evalBinaryExpr(exp *ast.BinaryExpr, params Parameter) (reflect.Value, error) {
	lhs, err := eval(exp.X, params)
	if err != nil {
		return reflect.Value{}, err
	}
	if lhs.Kind() == reflect.Func {
		return evalFunc(lhs, exp, params), nil
	}
	binaryExprExecutor, err := expr.FromToken(exp.Op)
	if err != nil {
		return reflect.Value{}, err
	}

	x := func() (reflect.Value, error) { return lhs, nil }

	// for lazy evaluation
	y := func() (reflect.Value, error) { return eval(exp.Y, params) }
	return binaryExprExecutor.Exec(x, y)
}
