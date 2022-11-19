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

// deepFieldByIndex get field by index
func sliceElem(rv reflect.Value) reflect.Value {
	return reflect.New(rv.Elem().Type().Elem()).Elem()
}
