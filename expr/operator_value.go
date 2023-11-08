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

package expr

import (
	"github.com/eatmoreapple/juice/internal/reflectlite"
	"reflect"
)

// OperatorExpr represents an operator expression.
type OperatorExpr int

const (
	Add  OperatorExpr = iota // +
	Sub                      // -
	Mul                      // *
	Quo                      // /
	Rem                      // %
	And                      // &
	Land                     // &&
	Or                       // |
	Lor                      // ||
	Eq                       // ==
	Ne                       // !=
	Lt                       // <
	Le                       // <=
	Gt                       // >
	Ge                       // >=
)

func (e OperatorExpr) String() string {
	switch e {
	case Add:
		return "+"
	case Sub:
		return "-"
	case Mul:
		return "*"
	case Quo:
		return "/"
	case Rem:
		return "%"
	case And:
		return "&"
	case Land:
		return "&&"
	case Or:
		return "|"
	case Lor:
		return "||"
	case Eq:
		return "=="
	case Ne:
		return "!="
	case Lt:
		return "<"
	case Le:
		return "<="
	case Gt:
		return ">"
	case Ge:
		return ">="
	default:
		return ""
	}
}

// OperationError represents an operation error between two values.
type OperationError struct {
	left, right reflect.Value
	operator    string
}

// Error implements errors interface.
func (c OperationError) Error() string {
	return "invalid operation " + c.operator + " for " + c.left.Kind().String() + " and " + c.right.Kind().String()
}

// NewOperationError creates a new OperationError.
func NewOperationError(left, right reflect.Value, operator string) error {
	return &OperationError{left: left, right: right, operator: operator}
}

// Operator defines an interface for operators.
type Operator interface {

	// Operate performs an operation between two values.
	Operate(left, right reflect.Value) (reflect.Value, error)
}

// IntOperator represents an integer operator.
type IntOperator struct {
	OperatorExpr
}

// Operate implements Operator interface.
func (o IntOperator) Operate(left, right reflect.Value) (reflect.Value, error) {
	left, right = reflectlite.Unwrap(left), reflectlite.Unwrap(right)
	if !isInt(left) || !isInt(right) {
		return reflect.Value{}, NewOperationError(left, right, o.OperatorExpr.String())
	}
	switch o.OperatorExpr {
	case Add:
		return reflect.ValueOf(left.Int() + right.Int()), nil
	case Sub:
		return reflect.ValueOf(left.Int() - right.Int()), nil
	case Mul:
		return reflect.ValueOf(left.Int() * right.Int()), nil
	case Quo:
		return reflect.ValueOf(left.Int() / right.Int()), nil
	case Rem:
		return reflect.ValueOf(left.Int() % right.Int()), nil
	case And:
		return reflect.ValueOf(left.Int() & right.Int()), nil
	case Land:
		return reflect.ValueOf(left.Int() != 0 && right.Int() != 0), nil
	case Or:
		return reflect.ValueOf(left.Int() | right.Int()), nil
	case Lor:
		return reflect.ValueOf(left.Int() != 0 || right.Int() != 0), nil
	case Eq:
		return reflect.ValueOf(left.Int() == right.Int()), nil
	case Ne:
		return reflect.ValueOf(left.Int() != right.Int()), nil
	case Lt:
		return reflect.ValueOf(left.Int() < right.Int()), nil
	case Le:
		return reflect.ValueOf(left.Int() <= right.Int()), nil
	case Gt:
		return reflect.ValueOf(left.Int() > right.Int()), nil
	case Ge:
		return reflect.ValueOf(left.Int() >= right.Int()), nil
	default:
		return invalidValue, NewOperationError(left, right, o.OperatorExpr.String())
	}
}

// UintOperator represents an unsigned integer operator.
type UintOperator struct {
	OperatorExpr
}

// Operate implements Operator interface.
func (o UintOperator) Operate(left, right reflect.Value) (reflect.Value, error) {
	left, right = reflectlite.Unwrap(left), reflectlite.Unwrap(right)
	if !isUint(left) || !isUint(right) {
		return reflect.Value{}, NewOperationError(left, right, o.OperatorExpr.String())
	}
	switch o.OperatorExpr {
	case Add:
		return reflect.ValueOf(left.Uint() + right.Uint()), nil
	case Sub:
		return reflect.ValueOf(left.Uint() - right.Uint()), nil
	case Mul:
		return reflect.ValueOf(left.Uint() * right.Uint()), nil
	case Quo:
		return reflect.ValueOf(left.Uint() / right.Uint()), nil
	case Rem:
		return reflect.ValueOf(left.Uint() % right.Uint()), nil
	case And:
		return reflect.ValueOf(left.Uint() & right.Uint()), nil
	case Land:
		return reflect.ValueOf(left.Uint() != 0 && right.Uint() != 0), nil
	case Or:
		return reflect.ValueOf(left.Uint() | right.Uint()), nil
	case Lor:
		return reflect.ValueOf(left.Uint() != 0 || right.Uint() != 0), nil
	case Eq:
		return reflect.ValueOf(left.Uint() == right.Uint()), nil
	case Ne:
		return reflect.ValueOf(left.Uint() != right.Uint()), nil
	case Lt:
		return reflect.ValueOf(left.Uint() < right.Uint()), nil
	case Le:
		return reflect.ValueOf(left.Uint() <= right.Uint()), nil
	case Gt:
		return reflect.ValueOf(left.Uint() > right.Uint()), nil
	case Ge:
		return reflect.ValueOf(left.Uint() >= right.Uint()), nil
	default:
		return invalidValue, NewOperationError(left, right, o.OperatorExpr.String())
	}
}

// FloatOperator represents a float operator.
type FloatOperator struct {
	OperatorExpr
}

// Operate implements Operator interface.
func (o FloatOperator) Operate(left, right reflect.Value) (reflect.Value, error) {
	left, right = reflectlite.Unwrap(left), reflectlite.Unwrap(right)
	if !isFloat(left) || !isFloat(right) {
		return reflect.Value{}, NewOperationError(left, right, o.OperatorExpr.String())
	}
	switch o.OperatorExpr {
	case Add:
		return reflect.ValueOf(left.Float() + right.Float()), nil
	case Sub:
		return reflect.ValueOf(left.Float() - right.Float()), nil
	case Mul:
		return reflect.ValueOf(left.Float() * right.Float()), nil
	case Quo:
		return reflect.ValueOf(left.Float() / right.Float()), nil
	case Rem:
		return reflect.ValueOf(float64(int64(left.Float()) % int64(right.Float()))), nil
	case And:
		return reflect.ValueOf(float64(int64(left.Float()) & int64(right.Float()))), nil
	case Land:
		return reflect.ValueOf(left.Float() != 0 && right.Float() != 0), nil
	case Or:
		return reflect.ValueOf(float64(int64(left.Float()) | int64(right.Float()))), nil
	case Lor:
		return reflect.ValueOf(left.Float() != 0 || right.Float() != 0), nil
	case Eq:
		return reflect.ValueOf(left.Float() == right.Float()), nil
	case Ne:
		return reflect.ValueOf(left.Float() != right.Float()), nil
	case Lt:
		return reflect.ValueOf(left.Float() < right.Float()), nil
	case Le:
		return reflect.ValueOf(left.Float() <= right.Float()), nil
	case Gt:
		return reflect.ValueOf(left.Float() > right.Float()), nil
	case Ge:
		return reflect.ValueOf(left.Float() >= right.Float()), nil
	default:
		return invalidValue, NewOperationError(left, right, o.OperatorExpr.String())
	}
}

// StringOperator represents a string operator.
type StringOperator struct {
	OperatorExpr
}

// Operate implements Operator interface.
func (o StringOperator) Operate(left, right reflect.Value) (reflect.Value, error) {
	left, right = reflectlite.Unwrap(left), reflectlite.Unwrap(right)
	if !isString(left) || !isString(right) {
		return reflect.Value{}, NewOperationError(left, right, o.OperatorExpr.String())
	}
	switch o.OperatorExpr {
	case Add:
		return reflect.ValueOf(left.String() + right.String()), nil
	case Eq:
		return reflect.ValueOf(left.String() == right.String()), nil
	case Ne:
		return reflect.ValueOf(left.String() != right.String()), nil
	case Lt:
		return reflect.ValueOf(left.String() < right.String()), nil
	case Le:
		return reflect.ValueOf(left.String() <= right.String()), nil
	case Gt:
		return reflect.ValueOf(left.String() > right.String()), nil
	case Ge:
		return reflect.ValueOf(left.String() >= right.String()), nil
	default:
		return invalidValue, NewOperationError(left, right, o.OperatorExpr.String())
	}
}

// BoolOperator represents a boolean operator.
type BoolOperator struct {
	OperatorExpr
}

// Operate implements Operator interface.
func (o BoolOperator) Operate(left, right reflect.Value) (reflect.Value, error) {
	left, right = reflectlite.Unwrap(left), reflectlite.Unwrap(right)
	if !isBool(left) || !isBool(right) {
		return reflect.Value{}, NewOperationError(left, right, o.OperatorExpr.String())
	}
	switch o.OperatorExpr {
	case And:
		return reflect.ValueOf(left.Bool() && right.Bool()), nil
	case Land:
		return reflect.ValueOf(left.Bool() && right.Bool()), nil
	case Or:
		return reflect.ValueOf(left.Bool() || right.Bool()), nil
	case Lor:
		return reflect.ValueOf(left.Bool() || right.Bool()), nil
	case Eq:
		return reflect.ValueOf(left.Bool() == right.Bool()), nil
	case Ne:
		return reflect.ValueOf(left.Bool() != right.Bool()), nil
	default:
		return invalidValue, NewOperationError(left, right, o.OperatorExpr.String())
	}
}

// ComplexOperator represents a complex operator.
type ComplexOperator struct {
	OperatorExpr
}

// Operate implements Operator interface.
func (o ComplexOperator) Operate(left, right reflect.Value) (reflect.Value, error) {
	left, right = reflectlite.Unwrap(left), reflectlite.Unwrap(right)
	if !isComplex(left) || !isComplex(right) {
		return reflect.Value{}, NewOperationError(left, right, o.OperatorExpr.String())
	}
	switch o.OperatorExpr {
	case Add:
		return reflect.ValueOf(left.Complex() + right.Complex()), nil
	case Sub:
		return reflect.ValueOf(left.Complex() - right.Complex()), nil
	case Mul:
		return reflect.ValueOf(left.Complex() * right.Complex()), nil
	case Quo:
		return reflect.ValueOf(left.Complex() / right.Complex()), nil
	case Eq:
		return reflect.ValueOf(left.Complex() == right.Complex()), nil
	case Ne:
		return reflect.ValueOf(left.Complex() != right.Complex()), nil
	default:
		return invalidValue, NewOperationError(left, right, o.OperatorExpr.String())
	}
}

// InvalidTypeOperator represents a type operator.
type InvalidTypeOperator struct {
	OperatorExpr
}

// Operate implements Operator interface.
func (o InvalidTypeOperator) Operate(left, right reflect.Value) (result reflect.Value, err error) {
	if !right.IsValid() || !left.IsValid() {
		// fixme: find a better way to handle nil values
		defer func() {
			if r := recover(); r != nil {
				// ignore panic
				left, right = reflectlite.Unwrap(left), reflectlite.Unwrap(right)
				err = NewOperationError(left, right, o.OperatorExpr.String())
			}
		}()
		ok := bothNil(left, right)
		switch o.OperatorExpr {
		case Eq:
			return reflect.ValueOf(ok), nil
		case Ne:
			return reflect.ValueOf(!ok), nil
		}
	}
	left, right = reflectlite.Unwrap(left), reflectlite.Unwrap(right)
	return invalidValue, NewOperationError(left, right, o.OperatorExpr.String())
}

// GenericOperator represents a generic operator.
type GenericOperator struct {
	OperatorExpr
}

// Operate implements Operator interface.
func (o GenericOperator) Operate(left, right reflect.Value) (reflect.Value, error) {
	var operator Operator
	if !right.IsValid() || !left.IsValid() {
		operator = InvalidTypeOperator{OperatorExpr: o.OperatorExpr}
		return operator.Operate(left, right)
	}
	right, left = reflectlite.Unwrap(right), reflectlite.Unwrap(left)
	switch {
	case isInt(left):
		operator = IntOperator{OperatorExpr: o.OperatorExpr}
	case isUint(left):
		operator = UintOperator{OperatorExpr: o.OperatorExpr}
	case isFloat(left):
		operator = FloatOperator{OperatorExpr: o.OperatorExpr}
	case isString(left):
		operator = StringOperator{OperatorExpr: o.OperatorExpr}
	case isBool(left):
		operator = BoolOperator{OperatorExpr: o.OperatorExpr}
	case isComplex(left):
		operator = ComplexOperator{OperatorExpr: o.OperatorExpr}
	default:
		return invalidValue, NewOperationError(left, right, o.OperatorExpr.String())
	}
	return operator.Operate(left, right)
}
