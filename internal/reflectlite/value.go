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

type Value struct {
	reflect.Value
}

// Unwrap returns the value of the element if the type is a pointer type.
// Otherwise, it returns the value directly.
func (v Value) Unwrap() reflect.Value {
	value := v.Value
	for {
		switch {
		case value.Kind() == reflect.Ptr:
			value = value.Elem()
		case value.Kind() == reflect.Interface:
			value = value.Elem()
		default:
			return value
		}
	}
}

// NilAble returns true if the type can be nil.
// only chan, func, interface, map, ptr, slice can be nil.
func (v Value) NilAble() bool {
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return true
	}
	return false
}

// IndirectType returns the type of the element if the type is a pointer type.
// Otherwise, it returns the type directly.
func (v Value) IndirectType() reflect.Type {
	return v.Unwrap().Type()
}

// IndirectKind returns the kind of the element if the type is a pointer type.
// Otherwise, it returns the kind of the type directly.
func (v Value) IndirectKind() reflect.Kind {
	return v.IndirectType().Kind()
}

// FindFieldFromTag returns the field value by tag name and tag value.
// It returns the zero Value if not found or the type is not struct.
func (v Value) FindFieldFromTag(tagName, tagValue string) Value {
	t := v.IndirectType()
	// only struct can have tag
	if t.Kind() != reflect.Struct {
		return Value{}
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if tag := field.Tag.Get(tagName); tag == tagValue {
			return From(v.Field(i))
		}
	}
	return Value{}
}

// ValueOf returns a new Value initialized to the concrete value
// stored in the interface i. ValueOf(nil) returns the zero Value.
func ValueOf(v any) Value {
	return Value{reflect.ValueOf(v)}
}

// From returns a new Value initialized to the concrete value
func From(v reflect.Value) Value {
	return Value{v}
}
