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

func Eval(expr string, params Parameter) (reflect.Value, error) {
	exp, err := parser.ParseExpr(expr)
	if err != nil {
		return reflect.Value{}, &SyntaxError{err}
	}
	return eval(exp, params)
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
		return reflect.Value{}, errors.New("unsupported expression")
	}
}

func evalSliceExpr(exp *ast.SliceExpr, params Parameter) (reflect.Value, error) {
	value, err := eval(exp.X, params)
	if err != nil {
		return reflect.Value{}, err
	}
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
	// return the slice
	return value.Slice(low, high), nil
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
	index, err := eval(exp.Index, params)
	if err != nil {
		return reflect.Value{}, err
	}
	switch value.Kind() {
	case reflect.Array, reflect.Slice, reflect.String:
		return value.Index(int(index.Int())), nil
	case reflect.Map:
		return value.MapIndex(index), nil
	default:
		return reflect.Value{}, errors.New("unsupported index expression")
	}
}

func evalCallExpr(exp *ast.CallExpr, params Parameter) (reflect.Value, error) {
	fn, err := eval(exp.Fun, params)
	if err != nil {
		return reflect.Value{}, err
	}
	if fn.Kind() != reflect.Func {
		return reflect.Value{}, errors.New("unsupported call expression")
	}
	if fn.Type().NumIn() != len(exp.Args) {
		return reflect.Value{}, fmt.Errorf("invalid number of arguments: expected %d, got %d", fn.Type().NumIn(), len(exp.Args))
	}
	if fn.Type().NumOut() != 1 {
		return reflect.Value{}, fmt.Errorf("invalid number of return values: expected 1, got %d", fn.Type().NumOut())
	}
	var args []reflect.Value
	for i, arg := range exp.Args {
		value, err := eval(arg, params)
		if err != nil {
			return reflect.Value{}, err
		}
		// type conversion for function arguments
		in := fn.Type().In(i)
		if in.Kind() != value.Kind() {
			if !value.CanConvert(in) {
				return reflect.Value{}, fmt.Errorf("cannot convert %s to %s", value.Type().Name(), in.Name())
			}
			value = value.Convert(in)
		}
		args = append(args, value)
	}
	return fn.Call(args)[0], nil
}

func evalSelectorExpr(exp *ast.SelectorExpr, params Parameter) (reflect.Value, error) {
	x, err := eval(exp.X, params)
	if err != nil {
		return reflect.Value{}, err
	}
	if x.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("invalid selector expression: %s", exp.Sel.Name)
	}
	return x.FieldByName(exp.Sel.Name), nil
}

func evalIdent(exp *ast.Ident, params Parameter) (reflect.Value, error) {
	if fn, ok := builtins[exp.Name]; ok {
		return fn, nil
	}
	value, ok := params.Get(exp.Name)
	if !ok {
		return reflect.Value{}, fmt.Errorf("undefined identifier: %s", exp.Name)
	}
	for value.Kind() == reflect.Interface {
		value = value.Elem()
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
	switch right.Kind() {
	case left.Kind():
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
	value, err := leq(right, left)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(!value.Bool()), nil
}

// geq returns true if right >= left.
func geq(right, left reflect.Value) (reflect.Value, error) {
	value, err := lss(right, left)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(!value.Bool()), nil
}

// land returns the logical and of the two values.
func land(right, left reflect.Value) (reflect.Value, error) {
	if right.Kind() == reflect.Bool && left.Kind() == reflect.Bool {
		return reflect.ValueOf(right.Bool() && left.Bool()), nil
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported land expression: %v, %v", right.Kind(), left.Kind())
}

// lor returns the logical or of the two values.
func lor(right, left reflect.Value) (reflect.Value, error) {
	if right.Kind() == reflect.Bool && left.Kind() == reflect.Bool {
		return reflect.ValueOf(right.Bool() || left.Bool()), nil
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported lor expression: %v, %v", right.Kind(), left.Kind())
}

// add returns the sum of the two values.
func add(right, left reflect.Value) (reflect.Value, error) {
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
	if right.Kind() == reflect.Bool && left.Kind() == right.Kind() {
		return reflect.ValueOf(right.Bool() && left.Bool()), nil
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported and expression: %v, %v", right.Kind(), left.Kind())
}

// or returns true if either right or left is true.
func or(right, left reflect.Value) (reflect.Value, error) {
	if right.Kind() == reflect.Bool && left.Kind() == right.Kind() {
		return reflect.ValueOf(right.Bool() || left.Bool()), nil
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported or expression: %v, %v", right.Kind(), left.Kind())
}
