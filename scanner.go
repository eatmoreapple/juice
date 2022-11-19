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

// one scan one row to given entity
func one(rows *sql.Rows, rv reflect.Value, mapper ResultMap) error {

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

		var dest = make([]any, len(columns))

		for index, column := range columns {
			// try to find field by column name
			field, ok, err := mapper.ColumnValue(rv, column)
			if err != nil {
				return err
			}
			if ok {
				dest[index] = field.Addr().Interface()
			} else {
				dest[index] = new(interface{})
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

// many scan rows to given entity slice
func many(rows *sql.Rows, rv reflect.Value, mapper ResultMap) error {

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

		// now, we start scan rows
		for rows.Next() {

			// make a new element of slice
			rv := reflect.New(el)

			// get the Value of element
			el := rv.Elem()

			// dest is the slice of interface which will be passed to rows.Scan
			var dest = make([]any, len(columns))

			// for each column, check if it's in indexMapping
			for index, column := range columns {
				// try to find the field in indexMapping
				field, ok, err := mapper.ColumnValue(el, column)
				if err != nil {
					return err
				}
				if ok {
					dest[index] = field.Addr().Interface()
				} else {
					dest[index] = new(interface{})
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

// Bind sql.Rows to given entity with default mapper
func Bind(rows *sql.Rows, v any) error {
	return BindWithResultMap(rows, v, nil)
}

// BindWithResultMap bind sql.Rows to given entity with given ResultMap
func BindWithResultMap(rows *sql.Rows, v any, mapper ResultMap) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return errors.New("v must be a pointer")
	}
	return bind(rows, rv, mapper)
}

// bind cover sql.Rows to given entity
// dest can be a pointer to a struct, a pointer to a slice of struct, or a pointer to a slice of any type.
// rows won't be closed when the function returns.
func bind(rows *sql.Rows, rv reflect.Value, mapper ResultMap) (err error) {
	if kd := reflect.Indirect(rv).Kind(); kd == reflect.Slice || kd == reflect.Array {
		return bindList(rows, rv, mapper)
	}
	return bindOne(rows, rv, mapper)
}

// bindOne cover sql.Rows to given entity
func bindOne(rows *sql.Rows, rv reflect.Value, mapper ResultMap) (err error) {
	if mapper == nil {
		// try to get mapper from entity
		mapper, err = newKeyValueResultMap(rv.Elem())
		if err != nil {
			return err
		}
	}
	return one(rows, rv, mapper)
}

// bindList cover sql.Rows to given entity slice
func bindList(rows *sql.Rows, rv reflect.Value, mapper ResultMap) (err error) {
	if mapper == nil {
		// try to get mapper from entity
		mapper, err = newIndexResultMap(sliceElem(rv))
		if err != nil {
			return err
		}
	}
	return many(rows, rv, mapper)
}

// Binder bind sql.Rows to dest
type Binder interface {
	// Scan sql.Rows to dest
	// dest can be a pointer to a struct, a pointer to a slice of struct, or a pointer to a slice of any type.
	Scan(v any) error
}

// rowsBinder is a wrapper of sql.Rows
// rowsBinder implements Binder
type rowsBinder struct {
	rows   *sql.Rows
	mapper ResultMap
}

// Scan implement Binder.Scan
func (r *rowsBinder) Scan(v any) error {
	defer func() { _ = r.rows.Close() }()
	return BindWithResultMap(r.rows, v, r.mapper)
}
