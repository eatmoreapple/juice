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
func deepFieldByIndex(rv reflect.Value, index []int) reflect.Value {
	for _, i := range index {
		rv = rv.Field(i)
	}
	return rv
}
