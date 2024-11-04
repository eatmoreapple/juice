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

package juice

import (
	"database/sql"
	"reflect"
	"time"
)

var (
	// scannerType is the reflect.Type of sql.Scanner
	// nolint:unused
	scannerType = reflect.TypeOf((*sql.Scanner)(nil)).Elem()

	// timeType is the reflect.Type of time.Time
	timeType = reflect.TypeOf((*time.Time)(nil)).Elem()
)

func bindWithResultMap(rows *sql.Rows, v any, resultMap ResultMap) error {
	if v == nil {
		return ErrNilDestination
	}
	if rows == nil {
		return ErrNilRows
	}
	// get reflect.Value of v
	rv := reflect.ValueOf(v)

	if rv.Kind() != reflect.Ptr {
		return ErrPointerRequired
	}
	// get default mapper
	if resultMap == nil {
		if kd := reflect.Indirect(rv).Kind(); kd == reflect.Slice {
			resultMap = MultiRowsResultMap{}
		} else {
			resultMap = SingleRowResultMap{}
		}
	}
	return resultMap.MapTo(rv, rows)
}

// BindWithResultMap bind sql.Rows to given entity with given ResultMap
// bind cover sql.Rows to given entity
// dest can be a pointer to a struct, a pointer to a slice of struct, or a pointer to a slice of any type.
// rows won't be closed when the function returns.
func BindWithResultMap[T any](rows *sql.Rows, resultMap ResultMap) (result T, err error) {
	// ptr is the pointer of the result, it is the destination of the binding.
	var ptr any = &result

	if _type := reflect.TypeOf(result); _type.Kind() == reflect.Ptr {
		// if the result is a pointer, create a new instance of the element.
		// you'd better not use a nil pointer as the result.
		result = reflect.New(_type.Elem()).Interface().(T)
		ptr = result
	}
	err = bindWithResultMap(rows, ptr, resultMap)
	return
}

// Bind sql.Rows to given entity with default mapper
func Bind[T any](rows *sql.Rows) (result T, err error) {
	return BindWithResultMap[T](rows, nil)
}

// List converts sql.Rows to a slice of the given entity type.
// If there are no rows, it will return an empty slice.
//
// Differences between List and Bind:
// - List always returns a slice, even if there is only one row.
// - Bind always returns the entity of the given type.
//
// Bind is more flexible; you can use it to bind a single row to a struct, a slice of structs, or a slice of any type.
// However, if you are sure that the result will be a slice, you can use List. It could be faster than Bind.
func List[T any](rows *sql.Rows) (result []T, err error) {
	var multiRowsResultMap MultiRowsResultMap

	element := reflect.TypeOf((*T)(nil)).Elem()

	// using reflect.New to create a new instance of the element is a very time-consuming operation.
	// if the element is not a pointer, we can create a new instance of it directly.
	if element.Kind() != reflect.Ptr {
		multiRowsResultMap.New = func() reflect.Value { return reflect.ValueOf(new(T)) }
	}

	err = multiRowsResultMap.MapTo(reflect.ValueOf(&result), rows)
	return result, err
}
