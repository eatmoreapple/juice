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

package reflectlite

import "reflect"

// Unwrap returns the value of the element if the type is a pointer or interface type.
func Unwrap(value reflect.Value) reflect.Value {
	for {
		switch value.Kind() {
		case reflect.Ptr, reflect.Interface:
			value = value.Elem()
		default:
			return value
		}
	}
}

// NilAble returns true if the type can be nil.
func NilAble(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer, reflect.Invalid:
		return true
	default:
	}
	return false
}

func IndirectKind(v reflect.Value) reflect.Kind {
	return IndirectType(v.Type()).Kind()
}

type Value struct {
	reflect.Value
}

// Unwrap returns the value of the element if the type is a pointer or interface type.
func (v Value) Unwrap() Value {
	value := Unwrap(v.Value)
	return Value{value}
}

// NilAble returns true if the type can be nil.
func (v Value) NilAble() bool {
	return NilAble(v.Value)
}

// IndirectType returns the type of the element if the type is a pointer type.
// Otherwise, it returns the type directly.
func (v Value) IndirectType() reflect.Type {
	return IndirectType(v.Type())
}

// IndirectKind returns the kind of the element if the type is a pointer type.
// Otherwise, it returns the kind of the type directly.
func (v Value) IndirectKind() reflect.Kind {
	return IndirectKind(v.Value)
}

// FindFieldFromTag returns the field value by tag name and tag value.
// It returns the zero Value if not found or the type is not struct.
func (v Value) FindFieldFromTag(tagName, tagValue string) Value {
	if v.Kind() != reflect.Struct {
		return Value{}
	}
	value, _ := findFieldFromTag(v, tagName, tagValue)
	return value
}

func findFieldFromTag(value Value, tagName, tagValue string) (Value, bool) {
	kind := value.IndirectType()
	for i := 0; i < kind.NumField(); i++ {
		field := kind.Field(i)
		if field.Type.Kind() == reflect.Struct && field.Tag.Get(tagName) == "" {
			if v, ok := findFieldFromTag(ValueFrom(value.Field(i)), tagName, tagValue); ok {
				return v, ok
			} else {
				continue
			}
		}
		if tag := field.Tag.Get(tagName); tag == tagValue {
			return ValueFrom(value.Field(i)), true
		}
	}
	return Value{}, false
}

// GetFieldIndexesFromTag returns the field indexes by tag name and tag value.
func (v Value) GetFieldIndexesFromTag(tagName, tagValue string) ([]int, bool) {
	return TypeFrom(v.IndirectType()).GetFieldIndexesFromTag(tagName, tagValue)
}

// ValueOf returns a new Value initialized to the concrete value
// stored in the interface i. ValueOf(nil) returns the zero Value.
func ValueOf(v any) Value {
	return Value{reflect.ValueOf(v)}
}

// ValueFrom returns a new Value initialized to the concrete value
func ValueFrom(v reflect.Value) Value {
	return Value{v}
}
