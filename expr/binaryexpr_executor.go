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
	"errors"
	"fmt"
	"github.com/eatmoreapple/juice/internal/reflectlite"
	"go/token"
	"reflect"
)

var (
	// nilValue represents the nil value
	nilValue = reflect.ValueOf(nil)

	// invalidValue represents the invalid value
	invalidValue = reflect.Value{}

	// trueValue represents the true value
	trueValue = reflect.ValueOf(true)

	// falseValue represents the false value
	falseValue = reflect.ValueOf(false)
)

// BinaryExprExecutor is the interface for binary expression executor
type BinaryExprExecutor interface {
	// Exec execute the binary expression
	// right is the right value of the binary expression
	// next is the function to get the left value of the binary expression
	// return the result of the binary expression
	Exec(left reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error)
}

type OperatorExecutor struct {
	Operator
}

func (c OperatorExecutor) Exec(left reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	right, err := next()
	if err != nil {
		return invalidValue, err
	}
	return c.Operator.Operate(left, right)
}

// EQLExprExecutor is the executor for ==
type EQLExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (EQLExprExecutor) Exec(left reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	var operator = GenericOperator{OperatorExpr: Eq}
	executor := OperatorExecutor{Operator: operator}
	return executor.Exec(left, next)
}

// NEQExprExecutor is the executor for !=
type NEQExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (NEQExprExecutor) Exec(left reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	var operator = GenericOperator{OperatorExpr: Ne}
	executor := OperatorExecutor{Operator: operator}
	return executor.Exec(left, next)
}

// LSSExprExecutor is the executor for <
type LSSExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (LSSExprExecutor) Exec(left reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	var operator = GenericOperator{OperatorExpr: Lt}
	executor := OperatorExecutor{Operator: operator}
	return executor.Exec(left, next)
}

// LEQExprExecutor is the executor for <=
type LEQExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (LEQExprExecutor) Exec(left reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	var operator = GenericOperator{OperatorExpr: Le}
	executor := OperatorExecutor{Operator: operator}
	return executor.Exec(left, next)
}

// GTRExprExecutor is the executor for >
type GTRExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (GTRExprExecutor) Exec(left reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	var operator = GenericOperator{OperatorExpr: Gt}
	executor := OperatorExecutor{Operator: operator}
	return executor.Exec(left, next)
}

// GEQExprExecutor is the executor for >=
type GEQExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (GEQExprExecutor) Exec(left reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	var operator = GenericOperator{OperatorExpr: Ge}
	executor := OperatorExecutor{Operator: operator}
	return executor.Exec(left, next)
}

// LANDExprExecutor is the executor for &&
type LANDExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (LANDExprExecutor) Exec(left reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	left = reflectlite.Unwrap(left)
	if left.Kind() != reflect.Bool {
		return invalidValue, fmt.Errorf("unsupported expression: %v", left.Kind())
	}
	if !left.Bool() {
		return left, nil
	}
	right, err := next()
	if err != nil {
		return invalidValue, err
	}
	right = reflectlite.Unwrap(right)
	if left.Kind() != reflect.Bool {
		return invalidValue, fmt.Errorf("unsupported expression: %v", left.Kind())
	}
	return right, nil
}

// LORExprExecutor is the executor for ||
type LORExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (LORExprExecutor) Exec(left reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	left = reflectlite.Unwrap(left)
	if left.Kind() != reflect.Bool {
		return invalidValue, fmt.Errorf("unsupported expression: %v", left.Kind())
	}
	if left.Bool() {
		return left, nil
	}
	right, err := next()
	if err != nil {
		return invalidValue, err
	}
	right = reflectlite.Unwrap(right)
	if right.Kind() != reflect.Bool {
		return invalidValue, fmt.Errorf("unsupported expression: %v", right.Kind())
	}
	return right, nil
}

// ADDExprExecutor is the executor for +
type ADDExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (ADDExprExecutor) Exec(left reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	var operator = GenericOperator{OperatorExpr: Add}
	executor := OperatorExecutor{Operator: operator}
	return executor.Exec(left, next)
}

// SUBExprExecutor is the executor for -
type SUBExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (SUBExprExecutor) Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	var operator = GenericOperator{OperatorExpr: Sub}
	executor := OperatorExecutor{Operator: operator}
	return executor.Exec(right, next)
}

// MULExprExecutor is the executor for *
type MULExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (MULExprExecutor) Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	var operator = GenericOperator{OperatorExpr: Mul}
	executor := OperatorExecutor{Operator: operator}
	return executor.Exec(right, next)
}

// QUOExprExecutor is the executor for /
type QUOExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (QUOExprExecutor) Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	var operator = GenericOperator{OperatorExpr: Quo}
	executor := OperatorExecutor{Operator: operator}
	return executor.Exec(right, next)
}

// REMExprExecutor is the executor for %
type REMExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (REMExprExecutor) Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	var operator = GenericOperator{OperatorExpr: Rem}
	executor := OperatorExecutor{Operator: operator}
	return executor.Exec(right, next)
}

// LPARENExprExecutor is the executor for (
type LPARENExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (LPARENExprExecutor) Exec(_ reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	return next()
}

// RPARENExprExecutor is the executor for )
// it just return the value
type RPARENExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (RPARENExprExecutor) Exec(right reflect.Value, _ func() (reflect.Value, error)) (reflect.Value, error) {
	return right, nil
}

type COMMENTExprExecutor struct{}

func (COMMENTExprExecutor) Exec(_ reflect.Value, _ func() (reflect.Value, error)) (reflect.Value, error) {
	return reflect.ValueOf(true), nil
}

// NOTExprExecutor is the executor for !
type NOTExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (NOTExprExecutor) Exec(_ reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	right, err := next()
	if err != nil {
		return invalidValue, err
	}
	return reflect.ValueOf(!right.Bool()), nil
}

// ANDExprExecutor is the executor for &&
type ANDExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (ANDExprExecutor) Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	right = reflectlite.Unwrap(right)
	if right.Kind() != reflect.Bool {
		return invalidValue, fmt.Errorf("unsupported and expression: %v", right.Kind())
	}
	if !right.Bool() {
		return right, nil
	}
	left, err := next()
	if err != nil {
		return invalidValue, err
	}
	left = reflectlite.Unwrap(left)
	if left.Kind() != reflect.Bool {
		return invalidValue, fmt.Errorf("unsupported and expression: %v", left.Kind())
	}
	return left, nil
}

// ORExprExecutor is the executor for ||
type ORExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (ORExprExecutor) Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	right = reflectlite.Unwrap(right)
	if right.Kind() != reflect.Bool {
		return invalidValue, fmt.Errorf("unsupported or expression: %v", right.Kind())
	}
	if right.Bool() {
		return right, nil
	}
	left, err := next()
	if err != nil {
		return invalidValue, err
	}
	left = reflectlite.Unwrap(left)
	if left.Kind() != reflect.Bool {
		return invalidValue, fmt.Errorf("unsupported or expression: %v", left.Kind())
	}
	return left, nil
}

// ErrUnsupportedBinaryExpr is the error that the binary expression is unsupported
var ErrUnsupportedBinaryExpr = errors.New("unsupported binary expression")

// FromToken returns the BinaryExprExecutor from the token
func FromToken(t token.Token) (BinaryExprExecutor, error) {
	var binaryExprExecutor BinaryExprExecutor
	switch t {
	case token.EQL:
		binaryExprExecutor = EQLExprExecutor{}
	case token.NEQ:
		binaryExprExecutor = NEQExprExecutor{}
	case token.LSS:
		binaryExprExecutor = LSSExprExecutor{}
	case token.LEQ:
		binaryExprExecutor = LEQExprExecutor{}
	case token.GTR:
		binaryExprExecutor = GTRExprExecutor{}
	case token.GEQ:
		binaryExprExecutor = GEQExprExecutor{}
	case token.LAND:
		binaryExprExecutor = LANDExprExecutor{}
	case token.LOR:
		binaryExprExecutor = LORExprExecutor{}
	case token.ADD:
		binaryExprExecutor = ADDExprExecutor{}
	case token.SUB:
		binaryExprExecutor = SUBExprExecutor{}
	case token.MUL:
		binaryExprExecutor = MULExprExecutor{}
	case token.QUO:
		binaryExprExecutor = QUOExprExecutor{}
	case token.REM:
		binaryExprExecutor = REMExprExecutor{}
	case token.LPAREN:
		binaryExprExecutor = LPARENExprExecutor{}
	case token.RPAREN:
		binaryExprExecutor = RPARENExprExecutor{}
	case token.COMMENT:
		binaryExprExecutor = COMMENTExprExecutor{}
	case token.NOT:
		binaryExprExecutor = NOTExprExecutor{}
	case token.AND:
		binaryExprExecutor = ANDExprExecutor{}
	case token.OR:
		binaryExprExecutor = ORExprExecutor{}
	default:
		return nil, ErrUnsupportedBinaryExpr
	}
	return binaryExprExecutor, nil
}
