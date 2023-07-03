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

package juice

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math/cmplx"
	"reflect"
	"strconv"
)

// Evaluator is an evaluator of the expression.
type Evaluator interface {
	// Parse parses the expression and returns the expression.
	Parse(expr string) (Expression, error)
}

// EvalValue is an alias of reflect.Value.
// for semantic.
type EvalValue = reflect.Value

// Expression is an expression which can be evaluated to a value.
type Expression interface {
	// Eval evaluates the expression and returns the value.
	Eval(params Parameter) (EvalValue, error)
}

// goEvaluator is an evaluator of the expression who uses the go/ast package.
type goEvaluator struct{}

// Parse parses the expression and returns the expression.
func (e *goEvaluator) Parse(expr string) (Expression, error) {
	exp, err := parser.ParseExpr(expr)
	if err != nil {
		return nil, &SyntaxError{err}
	}
	return &goExpression{exp}, nil
}

// goExpression is an expression who uses the go/ast package.
type goExpression struct {
	ast.Expr
}

// Eval evaluates the expression and returns the value.
func (e *goExpression) Eval(params Parameter) (EvalValue, error) {
	return eval(e.Expr, params)
}

var (
	// DefaultEvaluator is the default evaluator.
	// Reset it to change the default behavior.
	DefaultEvaluator Evaluator = &goEvaluator{}
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

// Eval is a shortcut of DefaultEvaluator.Parse(expr).Eval(params).
func Eval(expr string, params Parameter) (EvalValue, error) {
	expression, err := DefaultEvaluator.Parse(expr)
	if err != nil {
		return reflect.Value{}, err
	}
	return expression.Eval(params)
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

	value = unwrapValue(value)

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
	var max int
	if exp.Max != nil {
		max, err = strconv.Atoi(exp.Max.(*ast.BasicLit).Value)
		if err != nil {
			return reflect.Value{}, err
		}
	}
	return value.Slice3(low, high, max), nil
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
	value = unwrapValue(value)

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
		value = unwrapValue(value)
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
		// should i mark it unreachable?
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

	unwarned := unwrapValue(x)

	// check if the field name is exported
	isExported := token.IsExported(fieldOrTagOrMethodName)

	var result reflect.Value

	switch unwarned.Kind() {
	case reflect.Struct:
		// findFromTag is a closure function that tries to find the field from the field tag
		findFromTag := func() {
			tp := unwarned.Type()
			for i := 0; i < unwarned.NumField(); i++ {
				field := tp.Field(i)
				if field.Tag.Get(defaultParamKey) == fieldOrTagOrMethodName {
					result = unwarned.Field(i)
					break
				}
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

			// not a method either, try to find from the field tag,
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

func evalBinaryExpr(exp *ast.BinaryExpr, params Parameter) (reflect.Value, error) {
	lhs, err := eval(exp.X, params)
	if err != nil {
		return reflect.Value{}, err
	}

	if lhs.Kind() == reflect.Func {
		return evalFunc(lhs, exp, params), nil
	}

	rhs, err := eval(exp.Y, params)
	if err != nil {
		return reflect.Value{}, err
	}
	var exprFunc func(lhs, rhs reflect.Value) (reflect.Value, error)
	switch exp.Op {
	case token.EQL:
		exprFunc = eql
	case token.NEQ:
		exprFunc = neq
	case token.LSS:
		exprFunc = lss
	case token.LEQ:
		exprFunc = leq
	case token.GTR:
		exprFunc = gtr
	case token.GEQ:
		exprFunc = geq
	case token.LAND:
		exprFunc = land
	case token.LOR:
		exprFunc = lor
	case token.ADD:
		exprFunc = add
	case token.SUB:
		exprFunc = sub
	case token.MUL:
		exprFunc = mul
	case token.QUO:
		exprFunc = quo
	case token.REM:
		exprFunc = rem
	case token.LPAREN:
		exprFunc = lparen
	case token.RPAREN:
		exprFunc = rparen
	case token.COMMENT:
		exprFunc = comment
	case token.NOT:
		exprFunc = not
	case token.AND:
		exprFunc = and
	case token.OR:
		exprFunc = or
	default:
		return reflect.Value{}, errors.New("unsupported binary expression")
	}
	return exprFunc(lhs, rhs)
}

func comment(_ reflect.Value, _ reflect.Value) (reflect.Value, error) {
	return reflect.ValueOf(true), nil
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
	return fn.Call(args)[0]
}

// eql returns true if the left and right values are equal.
func eql(right, left reflect.Value) (reflect.Value, error) {
	// check if the values are valid
	if !right.IsValid() || !left.IsValid() {

		// if both values are invalid, they are equal
		if !right.IsValid() && !left.IsValid() {
			return reflect.ValueOf(true), nil
		}
		var valid = right
		if !right.IsValid() {
			valid = left
		}

		// if the invalid value is nil, the valid value is equal to nil
		if isNilAble(valid) {
			// nil value
			if valid.Equal(nilValue) {
				return reflect.ValueOf(true), nil
			}

			// unwrap interface value
			if valid.Kind() == reflect.Interface {
				valid = valid.Elem()
			}
			// nil value but not nil type
			return reflect.ValueOf(valid.IsNil()), nil
		}
		return reflect.ValueOf(false), fmt.Errorf("invalid operation: %s == %s", right.Kind(), left.Kind())
	}

	right, left = unwrapValue(right), unwrapValue(left)

	// check if the values are comparable
	switch right.Kind() {
	case left.Kind():
		// if they are same kind, use reflect.DeepEqual
		value := reflect.DeepEqual(right.Interface(), left.Interface())
		return reflect.ValueOf(value), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch {
		case reflect.Int <= left.Kind() && left.Kind() <= reflect.Int64:
			return reflect.ValueOf(right.Int() == left.Int()), nil
		case reflect.Uint <= left.Kind() && left.Kind() <= reflect.Uint64:
			return reflect.ValueOf(uint64(right.Int()) == left.Uint()), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch {
		case reflect.Int <= left.Kind() && left.Kind() <= reflect.Int64:
			return reflect.ValueOf(right.Uint() == uint64(left.Int())), nil
		case reflect.Uint <= left.Kind() && left.Kind() <= reflect.Uint64:
			return reflect.ValueOf(right.Uint() == left.Uint()), nil
		}
	case reflect.Float32, reflect.Float64:
		switch left.Kind() {
		case reflect.Float32, reflect.Float64:
			return reflect.ValueOf(right.Float() == left.Float()), nil
		}
	case reflect.Complex64, reflect.Complex128:
		switch left.Kind() {
		case reflect.Complex64, reflect.Complex128:
			return reflect.ValueOf(right.Complex() == left.Complex()), nil
		}
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported expression: %v, %v", right.Kind(), left.Kind())
}

// neq returns the result of a != b.
func neq(right, left reflect.Value) (reflect.Value, error) {
	value, err := eql(right, left)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(!value.Bool()), nil
}

// lss returns true if right < left.
func lss(right, left reflect.Value) (reflect.Value, error) {

	right, left = unwrapValue(right), unwrapValue(left)

	switch right.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch {
		case reflect.Int <= left.Kind() && left.Kind() <= reflect.Int64:
			return reflect.ValueOf(right.Int() < left.Int()), nil
		case reflect.Uint <= left.Kind() && left.Kind() <= reflect.Uint64:
			return reflect.ValueOf(uint64(right.Int()) < left.Uint()), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch {
		case reflect.Int <= left.Kind() && left.Kind() <= reflect.Int64:
			return reflect.ValueOf(right.Uint() < uint64(left.Int())), nil
		case reflect.Uint <= left.Kind() && left.Kind() <= reflect.Uint64:
			return reflect.ValueOf(right.Uint() < left.Uint()), nil
		}
	case reflect.Float32, reflect.Float64:
		switch left.Kind() {
		case reflect.Float32, reflect.Float64:
			return reflect.ValueOf(right.Float() < left.Float()), nil
		}
	case reflect.String:
		switch left.Kind() {
		case reflect.String:
			return reflect.ValueOf(right.String() < left.String()), nil
		}
	case reflect.Complex64, reflect.Complex128:
		switch left.Kind() {
		case reflect.Complex64, reflect.Complex128:
			return reflect.ValueOf(cmplx.Abs(right.Complex()) < cmplx.Abs(left.Complex())), nil
		}
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported expression: %v, %v", right.Kind(), left.Kind())
}

// leq returns true if right <= left.
func leq(right, left reflect.Value) (reflect.Value, error) {

	right, left = unwrapValue(right), unwrapValue(left)

	switch right.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch {
		case reflect.Int <= left.Kind() && left.Kind() <= reflect.Int64:
			return reflect.ValueOf(right.Int() <= left.Int()), nil
		case reflect.Uint <= left.Kind() && left.Kind() <= reflect.Uint64:
			return reflect.ValueOf(uint64(right.Int()) <= left.Uint()), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch {
		case reflect.Int <= left.Kind() && left.Kind() <= reflect.Int64:
			return reflect.ValueOf(right.Uint() <= uint64(left.Int())), nil
		case reflect.Uint <= left.Kind() && left.Kind() <= reflect.Uint64:
			return reflect.ValueOf(right.Uint() <= left.Uint()), nil
		}
	case reflect.Float32, reflect.Float64:
		switch left.Kind() {
		case reflect.Float32, reflect.Float64:
			return reflect.ValueOf(right.Float() <= left.Float()), nil
		}
	case reflect.String:
		switch left.Kind() {
		case reflect.String:
			return reflect.ValueOf(right.String() <= left.String()), nil
		}
	case reflect.Complex64, reflect.Complex128:
		switch left.Kind() {
		case reflect.Complex64, reflect.Complex128:
			return reflect.ValueOf(cmplx.Abs(right.Complex()) <= cmplx.Abs(left.Complex())), nil
		}
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported expression: %v, %v", right.Kind(), left.Kind())
}

// gtr returns true if right > left
func gtr(right, left reflect.Value) (reflect.Value, error) {
	right, left = unwrapValue(right), unwrapValue(left)

	value, err := leq(right, left)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(!value.Bool()), nil
}

// geq returns true if right >= left.
func geq(right, left reflect.Value) (reflect.Value, error) {
	right, left = unwrapValue(right), unwrapValue(left)
	value, err := lss(right, left)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(!value.Bool()), nil
}

// land returns the logical and of the two values.
func land(right, left reflect.Value) (reflect.Value, error) {
	right, left = unwrapValue(right), unwrapValue(left)
	if right.Kind() == reflect.Bool && left.Kind() == reflect.Bool {
		return reflect.ValueOf(right.Bool() && left.Bool()), nil
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported land expression: %v, %v", right.Kind(), left.Kind())
}

// lor returns the logical or of the two values.
func lor(right, left reflect.Value) (reflect.Value, error) {
	right, left = unwrapValue(right), unwrapValue(left)
	if right.Kind() == reflect.Bool && left.Kind() == reflect.Bool {
		return reflect.ValueOf(right.Bool() || left.Bool()), nil
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported lor expression: %v, %v", right.Kind(), left.Kind())
}

// add returns the sum of the two values.
func add(right, left reflect.Value) (reflect.Value, error) {
	right, left = unwrapValue(right), unwrapValue(left)
	switch right.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch {
		case reflect.Int <= left.Kind() && left.Kind() <= reflect.Int64:
			return reflect.ValueOf(right.Int() + left.Int()), nil
		case reflect.Uint <= left.Kind() && left.Kind() <= reflect.Uint64:
			return reflect.ValueOf(uint64(right.Int()) + left.Uint()), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch {
		case reflect.Int <= left.Kind() && left.Kind() <= reflect.Int64:
			return reflect.ValueOf(right.Uint() + uint64(left.Int())), nil
		case reflect.Uint <= left.Kind() && left.Kind() <= reflect.Uint64:
			return reflect.ValueOf(right.Uint() + left.Uint()), nil
		}
	case reflect.Float32, reflect.Float64:
		switch left.Kind() {
		case reflect.Float32, reflect.Float64:
			return reflect.ValueOf(right.Float() + left.Float()), nil
		}
	case reflect.String:
		switch left.Kind() {
		case reflect.String:
			return reflect.ValueOf(right.String() + left.String()), nil
		}
	case reflect.Complex64, reflect.Complex128:
		switch left.Kind() {
		case reflect.Complex64, reflect.Complex128:
			return reflect.ValueOf(right.Complex() + left.Complex()), nil
		}
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported expression: %v, %v", right.Kind(), left.Kind())
}

// sub returns the difference between right and left.
func sub(right, left reflect.Value) (reflect.Value, error) {
	right, left = unwrapValue(right), unwrapValue(left)
	switch right.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch {
		case reflect.Int <= left.Kind() && left.Kind() <= reflect.Int64:
			return reflect.ValueOf(right.Int() - left.Int()), nil
		case reflect.Uint <= left.Kind() && left.Kind() <= reflect.Uint64:
			return reflect.ValueOf(uint64(right.Int()) - left.Uint()), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch {
		case reflect.Int <= left.Kind() && left.Kind() <= reflect.Int64:
			return reflect.ValueOf(right.Uint() - uint64(left.Int())), nil
		case reflect.Uint <= left.Kind() && left.Kind() <= reflect.Uint64:
			return reflect.ValueOf(right.Uint() - left.Uint()), nil
		}
	case reflect.Float32, reflect.Float64:
		switch left.Kind() {
		case reflect.Float32, reflect.Float64:
			return reflect.ValueOf(right.Float() - left.Float()), nil
		}
	case reflect.Complex64, reflect.Complex128:
		switch left.Kind() {
		case reflect.Complex64, reflect.Complex128:
			return reflect.ValueOf(right.Complex() - left.Complex()), nil
		}
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported expression: %v, %v", right.Kind(), left.Kind())
}

// mul returns the product of right and left.
func mul(right, left reflect.Value) (reflect.Value, error) {
	right, left = unwrapValue(right), unwrapValue(left)
	switch right.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch {
		case reflect.Int <= left.Kind() && left.Kind() <= reflect.Int64:
			return reflect.ValueOf(right.Int() * left.Int()), nil
		case reflect.Uint <= left.Kind() && left.Kind() <= reflect.Uint64:
			return reflect.ValueOf(uint64(right.Int()) * left.Uint()), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch {
		case reflect.Int <= left.Kind() && left.Kind() <= reflect.Int64:
			return reflect.ValueOf(right.Uint() * uint64(left.Int())), nil
		case reflect.Uint <= left.Kind() && left.Kind() <= reflect.Uint64:
			return reflect.ValueOf(right.Uint() * left.Uint()), nil
		}
	case reflect.Float32, reflect.Float64:
		switch left.Kind() {
		case reflect.Float32, reflect.Float64:
			return reflect.ValueOf(right.Float() * left.Float()), nil
		}
	case reflect.Complex64, reflect.Complex128:
		switch left.Kind() {
		case reflect.Complex64, reflect.Complex128:
			return reflect.ValueOf(right.Complex() * left.Complex()), nil
		}
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported expression: %v, %v", right.Kind(), left.Kind())
}

// quo returns the quotient of right and left.
func quo(right, left reflect.Value) (reflect.Value, error) {
	right, left = unwrapValue(right), unwrapValue(left)
	switch right.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch {
		case reflect.Int <= left.Kind() && left.Kind() <= reflect.Int64:
			return reflect.ValueOf(right.Int() / left.Int()), nil
		case reflect.Uint <= left.Kind() && left.Kind() <= reflect.Uint64:
			return reflect.ValueOf(uint64(right.Int()) / left.Uint()), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch {
		case reflect.Int <= left.Kind() && left.Kind() <= reflect.Int64:
			return reflect.ValueOf(right.Uint() / uint64(left.Int())), nil
		case reflect.Uint <= left.Kind() && left.Kind() <= reflect.Uint64:
			return reflect.ValueOf(right.Uint() / left.Uint()), nil
		}
	case reflect.Float32, reflect.Float64:
		switch left.Kind() {
		case reflect.Float32, reflect.Float64:
			return reflect.ValueOf(right.Float() / left.Float()), nil
		}
	case reflect.Complex64, reflect.Complex128:
		switch left.Kind() {
		case reflect.Complex64, reflect.Complex128:
			return reflect.ValueOf(right.Complex() / left.Complex()), nil
		}
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported expression: %v, %v", right.Kind(), left.Kind())

}

// rem returns the remainder of a division operation.
func rem(right, left reflect.Value) (reflect.Value, error) {
	right, left = unwrapValue(right), unwrapValue(left)
	switch right.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch {
		case reflect.Int <= left.Kind() && left.Kind() <= reflect.Int64:
			return reflect.ValueOf(right.Int() % left.Int()), nil
		case reflect.Uint <= left.Kind() && left.Kind() <= reflect.Uint64:
			return reflect.ValueOf(uint64(right.Int()) % left.Uint()), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch {
		case reflect.Int <= left.Kind() && left.Kind() <= reflect.Int64:
			return reflect.ValueOf(right.Uint() % uint64(left.Int())), nil
		case reflect.Uint <= left.Kind() && left.Kind() <= reflect.Uint64:
			return reflect.ValueOf(right.Uint() % left.Uint()), nil
		}
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported expression: %v, %v", right.Kind(), left.Kind())
}

// land returns true if both right and left are true.
func lparen(_, left reflect.Value) (reflect.Value, error) {
	return left, nil
}

// land returns true if both right and left are true.
func rparen(right, _ reflect.Value) (reflect.Value, error) {
	return right, nil
}

// not returns true if right is false.
func not(_, left reflect.Value) (reflect.Value, error) {
	left = unwrapValue(left)
	if left.Kind() == reflect.Bool {
		return reflect.ValueOf(!left.Bool()), nil
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported not expression: %v", left.Kind())
}

// and returns true if both right and left are true.
// what's the difference between and land?
// land will evaluate left if right is false.
// but not and.
// for example:
//
//			1 + 1 == 2 && 1 + 1 == 3    // false
//		 	1 + 1 == 2 & 1 + 1 == 3     // it will return an error, cause 2 & 1 are not bool value.
//	     	(1 + 1 == 2) & (1 + 1 == 3) // this is ok.
func and(right, left reflect.Value) (reflect.Value, error) {
	right, left = unwrapValue(right), unwrapValue(left)
	if right.Kind() == reflect.Bool && left.Kind() == right.Kind() {
		return reflect.ValueOf(right.Bool() && left.Bool()), nil
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported and expression: %v, %v", right.Kind(), left.Kind())
}

// or returns true if either right or left is true.
func or(right, left reflect.Value) (reflect.Value, error) {
	right, left = unwrapValue(right), unwrapValue(left)
	if right.Kind() == reflect.Bool && left.Kind() == right.Kind() {
		return reflect.ValueOf(right.Bool() || left.Bool()), nil
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported or expression: %v, %v", right.Kind(), left.Kind())
}
