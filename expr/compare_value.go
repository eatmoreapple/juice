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
	"fmt"
	"github.com/eatmoreapple/juice/internal/reflectlite"
	"reflect"
)

type CompareOperationError struct {
	left, right reflect.Value
	expr        CompareExpr
}

// Error implements errors interface.
func (c CompareOperationError) Error() string {
	return "invalid operation " + c.expr.String() + " for " + c.left.Kind().String() + " and " + c.right.Kind().String()
}

func NewCompareOperationError(left, right reflect.Value, expr CompareExpr) error {
	return &CompareOperationError{left: left, right: right, expr: expr}
}

type CompareExpr int

const (
	eq CompareExpr = iota // ==
	ne                    // !=
	lt                    // <
	le                    // <=
	gt                    // >
	ge                    // >=
)

func (e CompareExpr) String() string {
	switch e {
	case eq:
		return "=="
	case ne:
		return "!="
	case lt:
		return "<"
	case le:
		return "<="
	case gt:
		return ">"
	case ge:
		return ">="
	default:
		return ""
	}
}

func isInt(r reflect.Value) bool {
	switch r.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	default:
		return false
	}
}

func isUint(r reflect.Value) bool {
	switch r.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func isFloat(r reflect.Value) bool {
	switch r.Kind() {
	case reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func isComplex(r reflect.Value) bool {
	switch r.Kind() {
	case reflect.Complex64, reflect.Complex128:
		return true
	default:
		return false
	}
}

type Comparer interface {
	Compare(left, right reflect.Value) (bool, error)
}

type IntValueComparer struct {
	CompareExpr
}

func (c IntValueComparer) Compare(left, right reflect.Value) (bool, error) {
	left, right = reflectlite.Unwrap(left), reflectlite.Unwrap(right)
	if !left.IsValid() || !right.IsValid() {
		return false, nil
	}
	if !isInt(right) || !isInt(left) {
		return false, NewCompareOperationError(left, right, c.CompareExpr)
	}
	switch c.CompareExpr {
	case eq:
		return left.Int() == right.Int(), nil
	case ne:
		return left.Int() != right.Int(), nil
	case lt:
		return left.Int() < right.Int(), nil
	case le:
		return left.Int() <= right.Int(), nil
	case gt:
		return left.Int() > right.Int(), nil
	case ge:
		return left.Int() >= right.Int(), nil
	default:
		return false, NewCompareOperationError(left, right, c.CompareExpr)
	}
}

type UintValueComparer struct {
	CompareExpr
}

func (c UintValueComparer) Compare(left, right reflect.Value) (bool, error) {
	right, left = reflectlite.Unwrap(right), reflectlite.Unwrap(left)
	if !left.IsValid() || !right.IsValid() {
		return false, nil
	}
	if !isUint(right) || !isUint(left) {
		return false, NewCompareOperationError(left, right, c.CompareExpr)
	}
	switch c.CompareExpr {
	case eq:
		return left.Uint() == right.Uint(), nil
	case ne:
		return left.Uint() != right.Uint(), nil
	case lt:
		return left.Uint() < right.Uint(), nil
	case le:
		return left.Uint() <= right.Uint(), nil
	case gt:
		return left.Uint() > right.Uint(), nil
	case ge:
		return left.Uint() >= right.Uint(), nil
	default:
		return false, NewCompareOperationError(left, right, c.CompareExpr)
	}
}

type FloatValueComparer struct {
	CompareExpr
}

func (c FloatValueComparer) Compare(left, right reflect.Value) (bool, error) {
	right, left = reflectlite.Unwrap(right), reflectlite.Unwrap(left)
	if !left.IsValid() || !right.IsValid() {
		return false, nil
	}
	if !isFloat(right) || !isFloat(left) {
		return false, NewCompareOperationError(left, right, c.CompareExpr)
	}
	switch c.CompareExpr {
	case eq:
		return left.Float() == right.Float(), nil
	case ne:
		return left.Float() != right.Float(), nil
	case lt:
		return left.Float() < right.Float(), nil
	case le:
		return left.Float() <= right.Float(), nil
	case gt:
		return left.Float() > right.Float(), nil
	case ge:
		return left.Float() >= right.Float(), nil
	default:
		return false, NewCompareOperationError(left, right, c.CompareExpr)
	}
}

type StringValueComparer struct {
	CompareExpr
}

func (c StringValueComparer) Compare(left, right reflect.Value) (bool, error) {
	right, left = reflectlite.Unwrap(right), reflectlite.Unwrap(left)
	if !left.IsValid() || !right.IsValid() {
		return false, nil
	}
	if left.Kind() != reflect.String || right.Kind() != reflect.String {
		return false, fmt.Errorf("invalid operation: %s == %s", right.Kind(), left.Kind())
	}
	switch c.CompareExpr {
	case eq:
		return left.String() == right.String(), nil
	case ne:
		return left.String() != right.String(), nil
	case lt:
		return left.String() < right.String(), nil
	case le:
		return left.String() <= right.String(), nil
	case gt:
		return left.String() > right.String(), nil
	case ge:
		return left.String() >= right.String(), nil
	default:
		return false, NewCompareOperationError(left, right, c.CompareExpr)
	}
}

type BoolValueComparer struct {
	CompareExpr
}

func (c BoolValueComparer) Compare(left, right reflect.Value) (bool, error) {
	right, left = reflectlite.Unwrap(right), reflectlite.Unwrap(left)
	if !left.IsValid() || !right.IsValid() {
		return false, nil
	}
	if left.Kind() != reflect.Bool || right.Kind() != reflect.Bool {
		return false, NewCompareOperationError(left, right, c.CompareExpr)
	}
	switch c.CompareExpr {
	case eq:
		return left.Bool() == right.Bool(), nil
	case ne:
		return left.Bool() != right.Bool(), nil
	default:
		return false, NewCompareOperationError(left, right, c.CompareExpr)
	}
}

type ComplexValueComparer struct {
	CompareExpr
}

func (c ComplexValueComparer) Compare(left, right reflect.Value) (bool, error) {
	right, left = reflectlite.Unwrap(right), reflectlite.Unwrap(left)
	if !left.IsValid() || !right.IsValid() {
		return false, nil
	}
	if !isComplex(left) || !isComplex(right) {
		return false, NewCompareOperationError(left, right, c.CompareExpr)
	}
	switch c.CompareExpr {
	case eq:
		return left.Complex() == right.Complex(), nil
	case ne:
		return left.Complex() != right.Complex(), nil
	default:
		return false, NewCompareOperationError(left, right, c.CompareExpr)
	}
}

type GenericValueComparer struct {
	CompareExpr
}

func (c GenericValueComparer) Compare(left, right reflect.Value) (bool, error) {
	var typeComparer Comparer
	left, right = reflectlite.Unwrap(left), reflectlite.Unwrap(right)
	switch {
	case isInt(right):
		typeComparer = IntValueComparer{CompareExpr: c.CompareExpr}
	case isUint(right):
		typeComparer = UintValueComparer{CompareExpr: c.CompareExpr}
	case isFloat(right):
		typeComparer = FloatValueComparer{CompareExpr: c.CompareExpr}
	case isComplex(right):
		typeComparer = ComplexValueComparer{CompareExpr: c.CompareExpr}
	case right.Kind() == reflect.String:
		typeComparer = StringValueComparer{CompareExpr: c.CompareExpr}
	case right.Kind() == reflect.Bool:
		typeComparer = BoolValueComparer{CompareExpr: c.CompareExpr}
	default:
		return false, NewCompareOperationError(left, right, c.CompareExpr)
	}
	return typeComparer.Compare(left, right)
}

type EqualComparer struct{}

func (c EqualComparer) Compare(left, right reflect.Value) (bool, error) {
	// check if the values are valid
	if !right.IsValid() || !left.IsValid() {

		// if both values are invalid, they are equal
		if !right.IsValid() && !left.IsValid() {
			return true, nil
		}
		var valid = right
		if !right.IsValid() {
			valid = left
		}

		// if the invalid value is nil, the valid value is equal to nil
		if reflectlite.NilAble(valid) {
			// nil value
			if valid.Equal(nilValue) {
				return true, nil
			}

			// unwrap interface value
			if valid.Kind() == reflect.Interface {
				valid = valid.Elem()
			}
			// nil value but not nil type
			return valid.IsNil(), nil
		}
		return false, NewCompareOperationError(left, right, eq)
	}

	right, left = reflectlite.Unwrap(right), reflectlite.Unwrap(left)

	if right.Kind() == left.Kind() {
		return reflect.DeepEqual(right.Interface(), left.Interface()), nil
	}
	var comparer = GenericValueComparer{CompareExpr: eq}
	return comparer.Compare(right, left)
}

type NotEqualComparer struct{}

func (c NotEqualComparer) Compare(left, right reflect.Value) (bool, error) {
	var comparer = GenericValueComparer{CompareExpr: ne}
	return comparer.Compare(left, right)
}

type LessThanComparer struct{}

func (c LessThanComparer) Compare(left, right reflect.Value) (bool, error) {
	var comparer = GenericValueComparer{CompareExpr: lt}
	return comparer.Compare(left, right)
}

type LessThanOrEqualComparer struct{}

func (c LessThanOrEqualComparer) Compare(left, right reflect.Value) (bool, error) {
	var comparer = GenericValueComparer{CompareExpr: le}
	return comparer.Compare(left, right)
}

type GreaterThanComparer struct{}

func (c GreaterThanComparer) Compare(left, right reflect.Value) (bool, error) {
	var comparer = GenericValueComparer{CompareExpr: gt}
	return comparer.Compare(left, right)
}

type GreaterThanOrEqualComparer struct{}

func (c GreaterThanOrEqualComparer) Compare(left, right reflect.Value) (bool, error) {
	var comparer = GenericValueComparer{CompareExpr: ge}
	return comparer.Compare(left, right)
}
