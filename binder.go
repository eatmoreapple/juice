package juice

import (
	"database/sql"
	"errors"
	"reflect"
)

var (
	// scannerType is the reflect.Type of sql.Scanner
	scannerType = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
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
