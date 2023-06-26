package juice

import (
	"reflect"
)

// kindIndirect returns the type of the element of the pointer type.
func kindIndirect(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// typeIndirect returns the type of the element of the pointer type.
func typeIndirect(v reflect.Value) reflect.Kind {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v.Kind()
}

// unwrapValue returns the value of the element of the pointer type.
func unwrapValue(v reflect.Value) reflect.Value {
	for {
		switch {
		case v.Kind() == reflect.Ptr:
			v = v.Elem()
		case v.Kind() == reflect.Interface:
			v = v.Elem()
		default:
			return v
		}
	}
}

// isNilAble returns true if the type can be nil.
func isNilAble(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return true
	}
	return false
}
