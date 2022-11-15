package juice

import (
	"database/sql"
	"errors"
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
		columnValueMapping := make(map[string]reflect.Value)
		var tag = new(columnTag)
		for i := 0; i < re.NumField(); i++ {
			field := re.Field(i)
			if field.Anonymous {
				continue
			}
			if column := field.Tag.Get("column"); column != "" {
				tag.parse(column)
				columnValueMapping[tag.Name] = rv.Field(i)
				tag.reset()
			}
		}

		var dest = make([]any, len(columns))

		for index, column := range columns {
			// many field in columnValueMapping first
			if field, ok := columnValueMapping[column]; ok {
				dest[index] = field.Addr().Interface()
			} else {
				fieldName := underlineToCamel(column)
				elField := rv.FieldByName(fieldName)
				if !elField.IsValid() || !elField.CanSet() {
					dest[index] = new(any)
				} else {
					dest[index] = elField.Addr().Interface()
				}
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
	if kd := rv.Elem().Kind(); kd != reflect.Slice && kd != reflect.Array {
		return errors.New("v must be a pointer to slice or array")
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
	for el.Kind() == reflect.Ptr {
		el = el.Elem()
	}

	// if it's a struct, scan rows to struct
	if el.Kind() == reflect.Struct {
		// get columns from rows
		columns, err := rows.Columns()
		if err != nil {
			return err
		}
		// column reflect.Value mapping
		columnValueMapping := make(map[string]int)
		var tag = new(columnTag)

		// get the field of struct
		// if the field has a tag named column, use the tag as column name
		// else use the field name as column name
		// then put the field index into columnValueMapping
		for i := 0; i < el.NumField(); i++ {
			field := el.Field(i)
			if field.Anonymous {
				continue
			}

			// get the tag of field
			if column := field.Tag.Get("column"); column != "" {
				tag.parse(column)
				columnValueMapping[tag.Name] = i
				tag.reset()
			}
		}

		var checked bool

		// now, we start scan rows
		for rows.Next() {

			// make a new element of slice
			rv := reflect.New(el)

			// get the reflect.Value of element
			el := rv.Elem()

			// dest is the slice of interface which will be passed to rows.Scan
			var dest = make([]any, len(columns))

			// for each column, check if it's in columnValueMapping
			for index, column := range columns {
				// many field in columnValueMapping first
				// try to find the field in columnValueMapping
				if fieldIndex, ok := columnValueMapping[column]; ok {
					dest[index] = el.Field(fieldIndex).Addr().Interface()
				} else {

					// here we can't find the field in columnValueMapping
					fieldName := underlineToCamel(column)

					// try to find the field in struct by field name
					elField := el.FieldByName(fieldName)

					// if the field is valid and can be set, use it
					// else use a pointer to interface
					if !elField.IsValid() || !elField.CanSet() {
						dest[index] = new(any)
					} else {
						dest[index] = elField.Addr().Interface()
					}
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
