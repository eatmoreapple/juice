package juice

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
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

func Eval(expr string, params map[string]reflect.Value) (reflect.Value, error) {
	exp, err := parser.ParseExpr(expr)
	if err != nil {
		return reflect.Value{}, &SyntaxError{err}
	}
	return eval(exp, params)
}

func eval(exp ast.Expr, params map[string]reflect.Value) (reflect.Value, error) {
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
	default:
		return reflect.Value{}, errors.New("unsupported expression")
	}
}

func evalSelectorExpr(exp *ast.SelectorExpr, params map[string]reflect.Value) (reflect.Value, error) {
	x, err := eval(exp.X, params)
	if err != nil {
		return reflect.Value{}, err
	}
	if x.Kind() != reflect.Struct {
		return reflect.Value{}, errors.New("unsupported selector expression")
	}
	return x.FieldByName(exp.Sel.Name), nil
}

func evalIdent(exp *ast.Ident, params Param) (reflect.Value, error) {
	value, ok := params.Get(exp.Name)
	if !ok {
		return reflect.Value{}, errors.New("undefined identifier")
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

func evalBinaryExpr(exp *ast.BinaryExpr, params map[string]reflect.Value) (reflect.Value, error) {
	lhs, err := eval(exp.X, params)
	if err != nil {
		return reflect.Value{}, err
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
	default:
		return reflect.Value{}, errors.New("unsupported binary expression")
	}
	return exprFunc(lhs, rhs)
}

func eql(right, left reflect.Value) (reflect.Value, error) {
	if right.Kind() == left.Kind() {
		return reflect.ValueOf(right.Interface() == left.Interface()), nil
	}
	switch right.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch left.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return reflect.ValueOf(right.Int() == left.Int()), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch left.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return reflect.ValueOf(right.Uint() == left.Uint()), nil
		}
	case reflect.Float32, reflect.Float64:
		switch left.Kind() {
		case reflect.Float32, reflect.Float64:
			return reflect.ValueOf(right.Float() == left.Float()), nil
		}
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported eql expression: %v, %v", right.Kind(), left.Kind())
}

func neq(right, left reflect.Value) (reflect.Value, error) {
	if right.Kind() == left.Kind() {
		return reflect.ValueOf(right.Interface() != left.Interface()), nil
	}
	switch right.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch left.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return reflect.ValueOf(right.Int() != left.Int()), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch left.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return reflect.ValueOf(right.Uint() != left.Uint()), nil
		}
	case reflect.Float32, reflect.Float64:
		switch left.Kind() {
		case reflect.Float32, reflect.Float64:
			return reflect.ValueOf(right.Float() != left.Float()), nil
		}
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported neq expression: %v, %v", right.Kind(), left.Kind())
}

func lss(right, left reflect.Value) (reflect.Value, error) {
	switch right.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch left.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return reflect.ValueOf(right.Int() < left.Int()), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch left.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
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
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported lss expression: %v, %v", right.Kind(), left.Kind())
}

func leq(right, left reflect.Value) (reflect.Value, error) {
	switch right.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch left.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return reflect.ValueOf(right.Int() <= left.Int()), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch left.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
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
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported leq expression: %v, %v", right.Kind(), left.Kind())
}

func gtr(right, left reflect.Value) (reflect.Value, error) {
	switch right.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch left.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return reflect.ValueOf(right.Int() > left.Int()), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch left.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return reflect.ValueOf(right.Uint() > left.Uint()), nil
		}
	case reflect.Float32, reflect.Float64:
		switch left.Kind() {
		case reflect.Float32, reflect.Float64:
			return reflect.ValueOf(right.Float() > left.Float()), nil
		}
	case reflect.String:
		switch left.Kind() {
		case reflect.String:
			return reflect.ValueOf(right.String() > left.String()), nil
		}
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported gtr expression: %v, %v", right.Kind(), left.Kind())
}

func geq(right, left reflect.Value) (reflect.Value, error) {
	switch right.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch left.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return reflect.ValueOf(right.Int() >= left.Int()), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch left.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return reflect.ValueOf(right.Uint() >= left.Uint()), nil
		}
	case reflect.Float32, reflect.Float64:
		switch left.Kind() {
		case reflect.Float32, reflect.Float64:
			return reflect.ValueOf(right.Float() >= left.Float()), nil
		}
	case reflect.String:
		switch left.Kind() {
		case reflect.String:
			return reflect.ValueOf(right.String() >= left.String()), nil
		}
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported geq expression: %v, %v", right.Kind(), left.Kind())
}

func land(right, left reflect.Value) (reflect.Value, error) {
	if right.Kind() == reflect.Bool && left.Kind() == reflect.Bool {
		return reflect.ValueOf(right.Bool() && left.Bool()), nil
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported land expression: %v, %v", right.Kind(), left.Kind())
}

func lor(right, left reflect.Value) (reflect.Value, error) {
	if right.Kind() == reflect.Bool && left.Kind() == reflect.Bool {
		return reflect.ValueOf(right.Bool() || left.Bool()), nil
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported lor expression: %v, %v", right.Kind(), left.Kind())
}

func add(right, left reflect.Value) (reflect.Value, error) {
	switch right.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch left.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return reflect.ValueOf(right.Int() + left.Int()), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch left.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
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
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported add expression: %v, %v", right.Kind(), left.Kind())
}

func sub(right, left reflect.Value) (reflect.Value, error) {
	switch right.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch left.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return reflect.ValueOf(right.Int() - left.Int()), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch left.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return reflect.ValueOf(right.Uint() - left.Uint()), nil
		}
	case reflect.Float32, reflect.Float64:
		switch left.Kind() {
		case reflect.Float32, reflect.Float64:
			return reflect.ValueOf(right.Float() - left.Float()), nil
		}
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported sub expression: %v, %v", right.Kind(), left.Kind())
}

func mul(right, left reflect.Value) (reflect.Value, error) {
	switch right.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch left.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return reflect.ValueOf(right.Int() * left.Int()), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch left.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return reflect.ValueOf(right.Uint() * left.Uint()), nil
		}
	case reflect.Float32, reflect.Float64:
		switch left.Kind() {
		case reflect.Float32, reflect.Float64:
			return reflect.ValueOf(right.Float() * left.Float()), nil
		}
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported mul expression: %v, %v", right.Kind(), left.Kind())
}

func quo(right, left reflect.Value) (reflect.Value, error) {
	switch right.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch left.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return reflect.ValueOf(right.Int() / left.Int()), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch left.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return reflect.ValueOf(right.Uint() / left.Uint()), nil
		}
	case reflect.Float32, reflect.Float64:
		switch left.Kind() {
		case reflect.Float32, reflect.Float64:
			return reflect.ValueOf(right.Float() / left.Float()), nil
		}
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported quo expression: %v, %v", right.Kind(), left.Kind())
}

func rem(right, left reflect.Value) (reflect.Value, error) {
	switch right.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch left.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return reflect.ValueOf(right.Int() % left.Int()), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch left.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return reflect.ValueOf(right.Uint() % left.Uint()), nil
		}
	}
	return reflect.ValueOf(false), fmt.Errorf("unsupported rem expression: %v, %v", right.Kind(), left.Kind())
}
