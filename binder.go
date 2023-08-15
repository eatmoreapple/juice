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
func Bind(rows *sql.Rows, v any) error {
	return BindWithResultMap(rows, v, nil)
}

// BindWithResultMap bind sql.Rows to given entity with given ResultMap
// bind cover sql.Rows to given entity
// dest can be a pointer to a struct, a pointer to a slice of struct, or a pointer to a slice of any type.
// rows won't be closed when the function returns.
func BindWithResultMap(rows *sql.Rows, v any, resultMap ResultMap) error {
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
