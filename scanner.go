package juice

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
)

// One convert sql.Rows to given entity
func One(rows *sql.Rows, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return errors.New("v must be a pointer")
	}
	return one(rows, rv)
}

var (
	// scannerType is the reflect.Type of sql.Scanner
	scannerType = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
)

func scanFields(rv reflect.Value) (map[string]reflect.Value, error) {
	var dest = make(map[string]reflect.Value)
	if rv.Kind() == reflect.Struct {
		for i := 0; i < rv.NumField(); i++ {
			field := rv.Field(i)

			// skip unexported field and sql.Scanner
			if !field.CanSet() || field.Type().Implements(scannerType) {
				continue
			}

			// is deep struct
			if field.Kind() == reflect.Struct {
				// recursive call
				mapping, err := scanFields(field)
				if err != nil {
					return nil, err
				}
				for k, v := range mapping {
					if _, ok := dest[k]; ok {
						return nil, fmt.Errorf("field name %s is unbiguous", k)
					}
					dest[k] = v
				}
			} else {
				// skip field with no tag
				tag := rv.Type().Field(i).Tag.Get("column")
				if tag == "" {
					continue
				}
				if _, ok := dest[tag]; ok {
					return nil, fmt.Errorf("field name %s is unbiguous", tag)
				}
				dest[tag] = field
			}
		}
	}
	return dest, nil
}

// scanIndex is a helper function to scan rows to given entity
// it will return a map of column name to index of the field
func scanIndex(rv reflect.Value) (map[string][]int, error) {
	var dest = make(map[string][]int)
	if rv.Kind() == reflect.Struct {
		for i := 0; i < rv.NumField(); i++ {
			field := rv.Field(i)

			// is deep struct
			if field.Kind() == reflect.Struct && !field.Type().Implements(scannerType) {

				mapping, err := scanIndex(field)
				if err != nil {
					return nil, err
				}
				for k, v := range mapping {
					if _, ok := dest[k]; ok {
						return nil, fmt.Errorf("field name %s is unbiguous", k)
					}
					dest[k] = append([]int{i}, v...)
				}
			} else {

				// skip field with no tag
				tag := rv.Type().Field(i).Tag.Get("column")
				if tag == "" {
					continue
				}

				if _, ok := dest[tag]; ok {
					return nil, fmt.Errorf("field name %s is unbiguous", tag)
				}
				dest[tag] = []int{i}
			}
		}
	}
	return dest, nil
}

// one scan one row to given entity
func one(rows *sql.Rows, rv reflect.Value) error {

	// get element type
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	re := rv.Type()

	for re.Kind() == reflect.Ptr {
		re = re.Elem()
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}

	var err error

	if rv.Kind() == reflect.Struct {

		columns, err := rows.Columns()
		if err != nil {
			return err
		}
		// column reflect.Value mapping
		valueMapping, err := scanFields(rv)
		if err != nil {
			return err
		}

		var dest = make([]any, len(columns))

		for index, column := range columns {
			// try to find field by column name
			if field, ok := valueMapping[column]; ok {
				dest[index] = field.Addr().Interface()
			} else {
				// if not found, use any type
				dest[index] = new(any)
			}
		}
		for _, dp := range dest {
			if _, ok := dp.(*sql.RawBytes); ok {
				return errors.New("sql: RawBytes isn't allowed on SQLRowScanner.One")
			}
		}
		err = rows.Scan(dest...)
	} else {
		err = rows.Scan(rv.Addr().Interface())
	}
	return err
}

// Many cover sql.Rows to given entity
func Many(rows *sql.Rows, v any) error {
	rv := reflect.ValueOf(v)

	// pointer required
	if rv.Kind() != reflect.Ptr {
		return errors.New("v must be a pointer")
	}

	// check if it's a slice or array
	if kd := rv.Elem().Kind(); kd != reflect.Slice {
		return errors.New("v must be a pointer to slice")
	}

	// check pass then call many
	return many(rows, rv)
}

// many scan rows to given entity slice
func many(rows *sql.Rows, rv reflect.Value) error {

	// rv must be a pointer to slice or array
	rv = rv.Elem()

	// get the element type of slice or array
	el := rv.Type().Elem()

	// if it's a pointer, get the element type of pointer
	isPtr := el.Kind() == reflect.Ptr

	// make a new slice of element type
	result := reflect.MakeSlice(rv.Type(), 0, 0)

	// get the element type of pointer
	el = kindIndirect(el)

	// if it's a struct, scan rows to struct
	if el.Kind() == reflect.Struct {
		// get columns from rows
		columns, err := rows.Columns()
		if err != nil {
			return err
		}

		var checked bool

		var indexMapping map[string][]int

		// now, we start scan rows
		for rows.Next() {

			// make a new element of slice
			rv := reflect.New(el)

			// get the Value of element
			el := rv.Elem()

			if indexMapping == nil {
				indexMapping, err = scanIndex(el)
				if err != nil {
					return err
				}
			}

			// dest is the slice of interface which will be passed to rows.Scan
			var dest = make([]any, len(columns))

			// for each column, check if it's in indexMapping
			for index, column := range columns {
				// try to find the field in indexMapping
				if fieldIndex, ok := indexMapping[column]; ok {
					dest[index] = deepFieldByIndex(el, fieldIndex).Addr().Interface()
				} else {
					// but we can't
					// just ignore it
					dest[index] = new(any)
				}
			}

			// check if dest has sql.RawBytes
			if !checked {
				for _, dp := range dest {
					if _, ok := dp.(*sql.RawBytes); ok {
						return errors.New("sql: RawBytes isn't allowed on Row.Scan")
					}
				}
				checked = true
			}

			// scan rows to dest
			if err = rows.Scan(dest...); err != nil {
				return err
			}

			// has error, return it
			if err = rows.Err(); err != nil {
				return err
			}

			// append the element to result
			if isPtr {
				result = reflect.Append(result, rv)
			} else {
				result = reflect.Append(result, el)
			}
		}
	} else {
		// TODO: support other types
		// does not support map、slice

		// if it's not a struct, scan rows to pointer of element type
		for rows.Next() {

			rv := reflect.New(el)

			el := reflect.Indirect(rv)

			// scan rows to pointer of element type
			if err := rows.Scan(rv.Interface()); err != nil {
				return err
			}

			if err := rows.Err(); err != nil {
				return err
			}

			// append the element to result
			if isPtr {
				result = reflect.Append(result, rv)
			} else {
				result = reflect.Append(result, el)
			}
		}
	}

	// set result to given entity
	rv.Set(result)

	return nil
}

// Bind is a wrapper of bind
func Bind(rows *sql.Rows, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return errors.New("v must be a pointer")
	}
	return bind(rows, rv)
}

// bind cover sql.Rows to given entity
// dest can be a pointer to a struct, a pointer to a slice of struct, or a pointer to a slice of any type.
// rows won't be closed when the function returns.
func bind(rows *sql.Rows, rv reflect.Value) error {
	if kd := reflect.Indirect(rv).Kind(); kd == reflect.Slice || kd == reflect.Array {
		return many(rows, rv)
	}
	return one(rows, rv)
}

// Binder bind sql.Rows to dest
type Binder interface {
	One(v any) error
	Many(v any) error
}

// rowsBinder is a wrapper of sql.Rows
// rowsBinder implements Binder
type rowsBinder struct {
	rows *sql.Rows
}

// One bind sql.Rows to dest
func (r *rowsBinder) One(v any) error {
	defer func() { _ = r.rows.Close() }()
	return One(r.rows, v)
}

// Many bind sql.Rows to dest
func (r *rowsBinder) Many(v any) error {
	defer func() { _ = r.rows.Close() }()
	return Many(r.rows, v)
}
