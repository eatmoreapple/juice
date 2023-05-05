package juice

import (
	"reflect"
)

func kindIndirect(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

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
