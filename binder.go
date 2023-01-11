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

// Binder bind sql.Rows to dest
type Binder interface {
	// Bind sql.Rows to dest
	// It can be a pointer to a struct, a pointer to a slice of struct, or a pointer to a slice of any type.
	Bind(v any) error
}

// rowsBinder is a wrapper of sql.Rows
// rowsBinder implements Binder
type rowsBinder struct {
	rows   *sql.Rows
	mapper ResultMap
}

// Bind implement Binder.Bind
func (r *rowsBinder) Bind(v any) error {
	defer func() { _ = r.rows.Close() }()
	if scanner, ok := v.(RowsScanner); ok {
		return scanner.ScanRows(r.rows)
	}
	return BindWithResultMap(r.rows, v, r.mapper)
}
