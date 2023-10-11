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
	"errors"
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

// Bind sql.Rows to given entity with default mapper
func Bind[T any](rows *sql.Rows) (result T, err error) {
	return BindWithResultMap[T](rows, nil)
}

func bindWithResultMap(rows *sql.Rows, v any, resultMap ResultMap) error {
	if v == nil {
		return errors.New("destination can not be nil")
	}
	if rows == nil {
		return errors.New("rows can not be nil")
	}
	// get reflect.Value of v
	rv := reflect.ValueOf(v)

	if rv.Kind() != reflect.Ptr {
		return errors.New("v must be a pointer")
	}
	// get default mapper
	if resultMap == nil {
		if kd := reflect.Indirect(rv).Kind(); kd == reflect.Slice {
			resultMap = RowsResultMap{}
		} else {
			resultMap = RowResultMap{}
		}
	}
	return resultMap.ResultTo(rv, rows)
}

// BindWithResultMap bind sql.Rows to given entity with given ResultMap
// bind cover sql.Rows to given entity
// dest can be a pointer to a struct, a pointer to a slice of struct, or a pointer to a slice of any type.
// rows won't be closed when the function returns.
func BindWithResultMap[T any](rows *sql.Rows, resultMap ResultMap) (result T, err error) {
	// ptr is the pointer of the result, it is the destination of the binding.
	var ptr any = &result

	rv := reflect.ValueOf(result)
	// if the result is a pointer, create a new instance of the element.
	// you'd better not use a nil pointer as the result.
	// for example:
	//     BindWithResultMap[*int](rows, nil) bad
	//     BindWithResultMap[int](rows, nil) good
	if rv.Kind() == reflect.Ptr {
		result = reflect.New(rv.Type().Elem()).Interface().(T)
		ptr = result
	}
	err = bindWithResultMap(rows, ptr, resultMap)
	return
}
