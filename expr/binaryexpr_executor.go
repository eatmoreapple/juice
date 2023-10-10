package expr

import (
	"fmt"
	"github.com/eatmoreapple/juice/internal/reflectlite"
	"math/cmplx"
	"reflect"
)

var nilValue = reflect.ValueOf(nil)

// BinaryExprExecutor is the interface for binary expression executor
type BinaryExprExecutor interface {
	// Exec execute the binary expression
	// right is the right value of the binary expression
	// next is the function to get the left value of the binary expression
	// return the result of the binary expression
	Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error)
}

// EQLExprExecutor is the executor for ==
type EQLExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (EQLExprExecutor) Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	left, err := next()
	if err != nil {
		return reflect.Value{}, err
	}
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
		if reflectlite.NilAble(valid) {
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

	right, left = reflectlite.Unwrap(right), reflectlite.Unwrap(left)

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

// NEQExprExecutor is the executor for !=
type NEQExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (NEQExprExecutor) Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	exe := EQLExprExecutor{}
	value, err := exe.Exec(right, next)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(!value.Bool()), nil
}

// LSSExprExecutor is the executor for <
type LSSExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (LSSExprExecutor) Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	left, err := next()
	if err != nil {
		return reflect.Value{}, err
	}
	right, left = reflectlite.Unwrap(right), reflectlite.Unwrap(left)

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

// LEQExprExecutor is the executor for <=
type LEQExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (LEQExprExecutor) Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	left, err := next()
	if err != nil {
		return reflect.Value{}, err
	}
	right, left = reflectlite.Unwrap(right), reflectlite.Unwrap(left)

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

// GTRExprExecutor is the executor for >
type GTRExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (GTRExprExecutor) Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	exe := LEQExprExecutor{}
	value, err := exe.Exec(right, next)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(!value.Bool()), nil
}

// GEQExprExecutor is the executor for >=
type GEQExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (GEQExprExecutor) Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	exe := LSSExprExecutor{}
	value, err := exe.Exec(right, next)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(!value.Bool()), nil
}

// LANDExprExecutor is the executor for &&
type LANDExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (LANDExprExecutor) Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	right = reflectlite.Unwrap(right)
	if right.Kind() != reflect.Bool {
		return reflect.Value{}, fmt.Errorf("unsupported expression: %v", right.Kind())
	}
	if !right.Bool() {
		return right, nil
	}
	left, err := next()
	if err != nil {
		return reflect.Value{}, err
	}
	left = reflectlite.Unwrap(left)
	if left.Kind() != reflect.Bool {
		return reflect.Value{}, fmt.Errorf("unsupported expression: %v", left.Kind())
	}
	return left, nil
}

// LORExprExecutor is the executor for ||
type LORExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (LORExprExecutor) Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	right = reflectlite.Unwrap(right)
	if right.Kind() != reflect.Bool {
		return reflect.Value{}, fmt.Errorf("unsupported expression: %v", right.Kind())
	}
	if right.Bool() {
		return right, nil
	}
	left, err := next()
	if err != nil {
		return reflect.Value{}, err
	}
	left = reflectlite.Unwrap(left)
	if left.Kind() != reflect.Bool {
		return reflect.Value{}, fmt.Errorf("unsupported expression: %v", left.Kind())
	}
	return left, nil
}

// ADDExprExecutor is the executor for +
type ADDExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (ADDExprExecutor) Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	left, err := next()
	if err != nil {
		return reflect.Value{}, err
	}
	right, left = reflectlite.Unwrap(right), reflectlite.Unwrap(left)
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

// SUBExprExecutor is the executor for -
type SUBExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (SUBExprExecutor) Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	left, err := next()
	if err != nil {
		return reflect.Value{}, err
	}
	right, left = reflectlite.Unwrap(right), reflectlite.Unwrap(left)
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

// MULExprExecutor is the executor for *
type MULExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (MULExprExecutor) Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	left, err := next()
	if err != nil {
		return reflect.Value{}, err
	}
	right, left = reflectlite.Unwrap(right), reflectlite.Unwrap(left)
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

// QUOExprExecutor is the executor for /
type QUOExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (QUOExprExecutor) Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	left, err := next()
	if err != nil {
		return reflect.Value{}, err
	}
	right, left = reflectlite.Unwrap(right), reflectlite.Unwrap(left)
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

// REMExprExecutor is the executor for %
type REMExprExecutor struct{}

// Exec execute the binary expression
// implement BinaryExprExecutor interface
func (REMExprExecutor) Exec(right reflect.Value, next func() (reflect.Value, error)) (reflect.Value, error) {
	left, err := next()
	if err != nil {
		return reflect.Value{}, err
	}
	right, left = reflectlite.Unwrap(right), reflectlite.Unwrap(left)
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
		return reflect.Value{}, err
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
		return reflect.Value{}, fmt.Errorf("unsupported and expression: %v", right.Kind())
	}
	if !right.Bool() {
		return right, nil
	}
	left, err := next()
	if err != nil {
		return reflect.Value{}, err
	}
	left = reflectlite.Unwrap(left)
	if left.Kind() != reflect.Bool {
		return reflect.Value{}, fmt.Errorf("unsupported and expression: %v", left.Kind())
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
		return reflect.Value{}, fmt.Errorf("unsupported or expression: %v", right.Kind())
	}
	if right.Bool() {
		return right, nil
	}
	left, err := next()
	if err != nil {
		return reflect.Value{}, err
	}
	left = reflectlite.Unwrap(left)
	if left.Kind() != reflect.Bool {
		return reflect.Value{}, fmt.Errorf("unsupported or expression: %v", left.Kind())
	}
	return left, nil
}
