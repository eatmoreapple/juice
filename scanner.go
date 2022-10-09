package juice

import (
	"database/sql"
	"errors"
	"reflect"
)

func One[T any](rows *sql.Rows, err error) (T, error) {
	var result T

	if err != nil {
		return result, err
	}

	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err = rows.Err(); err != nil {
			return result, err
		}
		return result, sql.ErrNoRows
	}

	rt := reflect.TypeOf(result)

	isPtr := rt.Kind() == reflect.Ptr

	if isPtr {
		rt = rt.Elem()
	}
	rv := reflect.New(rt)

	el := rv.Elem()

	if el.Kind() == reflect.Struct {

		columns, err := rows.Columns()
		if err != nil {
			return result, err
		}
		// column reflect.Value mapping
		columnValueMapping := make(map[string]reflect.Value)
		var tag = new(columnTag)
		for i := 0; i < rt.NumField(); i++ {
			field := rt.Field(i)
			if field.Anonymous {
				continue
			}
			if column := field.Tag.Get("column"); column != "" {
				tag.parse(column)
				columnValueMapping[tag.Name] = el.Field(i)
				tag.reset()
			}
		}

		var dest = make([]any, len(columns))
		for index, column := range columns {
			// find field in columnValueMapping first
			if field, ok := columnValueMapping[column]; ok {
				dest[index] = field.Addr().Interface()
			} else {
				fieldName := underlineToCamel(column)
				elField := el.FieldByName(fieldName)
				if !elField.IsValid() || !elField.CanSet() {
					dest[index] = new(any)
				} else {
					dest[index] = elField.Addr().Interface()
				}
			}
		}
		for _, dp := range dest {
			if _, ok := dp.(*sql.RawBytes); ok {
				return result, errors.New("sql: RawBytes isn't allowed on SQLRowScanner.One")
			}
		}

		if err = rows.Scan(dest...); err != nil {
			return result, err
		}

	} else {
		if err = rows.Scan(el.Addr().Interface()); err != nil {
			return result, err
		}
	}
	if isPtr {
		result = rv.Interface().(T)
	} else {
		result = el.Interface().(T)
	}

	// Make sure the query can be processed to completion with no errors.
	return result, rows.Close()
}

func List[T any](rows *sql.Rows, err error) ([]T, error) {

	var result = make([]T, 0)

	if err != nil {
		return result, err
	}

	defer func() { _ = rows.Close() }()

	var item T

	rt := reflect.TypeOf(item)

	isPtr := rt.Kind() == reflect.Ptr

	if isPtr {
		rt = rt.Elem()
	}

	if rt.Kind() == reflect.Struct {
		columns, err := rows.Columns()
		if err != nil {
			return nil, err
		}
		// column reflect.Value mapping
		columnValueMapping := make(map[string]int)
		var tag = new(columnTag)
		for i := 0; i < rt.NumField(); i++ {
			field := rt.Field(i)
			if field.Anonymous {
				continue
			}
			if column := field.Tag.Get("column"); column != "" {
				tag.parse(column)
				columnValueMapping[tag.Name] = i
				tag.reset()
			}
		}

		var checked bool

		for rows.Next() {
			rv := reflect.New(rt)

			el := rv.Elem()

			var dest = make([]any, len(columns))
			for index, column := range columns {
				// find field in columnValueMapping first
				if fieldIndex, ok := columnValueMapping[column]; ok {
					dest[index] = el.Field(fieldIndex).Addr().Interface()
				} else {
					fieldName := underlineToCamel(column)
					elField := el.FieldByName(fieldName)
					if !elField.IsValid() || !elField.CanSet() {
						dest[index] = new(any)
					} else {
						dest[index] = elField.Addr().Interface()
					}
				}
			}

			if !checked {
				for _, dp := range dest {
					if _, ok := dp.(*sql.RawBytes); ok {
						return nil, errors.New("sql: RawBytes isn't allowed on Row.Scan")
					}
				}
				checked = true
			}

			if err = rows.Scan(dest...); err != nil {
				return nil, err
			}

			if err = rows.Err(); err != nil {
				return nil, err
			}

			if isPtr {
				result = append(result, rv.Interface().(T))
			} else {
				result = append(result, rv.Elem().Interface().(T))
			}
		}
	} else {
		// TODO: support other types
		// does not support map、slice
		for rows.Next() {
			rv := reflect.New(rt)

			el := rv.Elem()
			if err = rows.Scan(el.Addr().Interface()); err != nil {
				return nil, err
			}

			if err = rows.Err(); err != nil {
				return nil, err
			}

			if isPtr {
				result = append(result, rv.Interface().(T))
			} else {
				result = append(result, rv.Elem().Interface().(T))
			}
		}
	}

	return result, rows.Close()
}

type Scanner[T any] interface {
	One() (T, error)
	Many() ([]T, error)
}

type rowsScanner[T any] struct {
	rows *sql.Rows
	err  error
}

func (r *rowsScanner[T]) One() (T, error) {
	return One[T](r.rows, r.err)
}

func (r *rowsScanner[T]) Many() ([]T, error) {
	return List[T](r.rows, r.err)
}
