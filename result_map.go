package juice

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
)

type ResultMap interface {
	ResultTo(rv reflect.Value, row *sql.Rows) error
}

// RowResultMap is a ResultMap that maps a row to a non-slice type.
type RowResultMap struct{}

// ResultTo implements ResultMapper interface.
func (RowResultMap) ResultTo(rv reflect.Value, rows *sql.Rows) error {
	if rv.Kind() != reflect.Ptr {
		return errors.New("result must be a pointer")
	}

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

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	if re.Kind() == reflect.Struct {

		parser := tagParser{tag: "column", columns: columns, value: rv}

		dest, err := parser.destPoint()
		if err != nil {
			return err
		}

		for _, dp := range dest {
			if _, ok := dp.(*sql.RawBytes); ok {
				return errors.New("sql: RawBytes isn't allowed on SQLRowScanner.One")
			}
		}
		err = rows.Scan(dest...)
	} else {
		if len(columns) > 1 {
			return errors.New("sql: too many columns in result")
		}
		err = rows.Scan(rv.Addr().Interface())
	}
	return err
}

// RowsResultMap is a ResultMap that maps a row to a slice type.
type RowsResultMap struct{}

// ResultTo implements ResultMapper interface.
func (RowsResultMap) ResultTo(rv reflect.Value, rows *sql.Rows) error {

	if rv.Kind() != reflect.Ptr {
		return errors.New("result must be a pointer")
	}

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

	// get columns from rows
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	// if it's a struct, scan rows to struct
	if el.Kind() == reflect.Struct {

		var checked bool

		var parser *tagParser

		// now, we start scan rows
		for rows.Next() {

			// make a new element of slice
			rv := reflect.New(el)

			// get the Value of element
			el := rv.Elem()

			if parser == nil {
				parser = &tagParser{tag: "column", columns: columns, value: el}
			}

			dest, err := parser.destPoint()

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

		if len(columns) > 1 {
			return errors.New("sql: too many columns in result")
		}

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

// resultMapTag implements ResultMapper interface
type resultMapTag struct {
	id              string
	results         resultGroup
	associations    associationGroup
	collectionGroup collectionGroup
	mapping         map[string][]string
}

// init initializes resultMapTag
func (r *resultMapTag) init() error {
	r.mapping = make(map[string][]string)

	// add results to mapping
	m, err := r.results.mapping()
	if err != nil {
		return err
	}

	// check if there is any duplicate column
	for k, v := range m {
		if _, ok := r.mapping[k]; ok {
			return fmt.Errorf("field mapping %s is unbiguous", k)
		}
		r.mapping[k] = append(r.mapping[k], v...)
	}

	// add associations to mapping
	m, err = r.associations.mapping()
	if err != nil {
		return err
	}

	// check if there is any duplicate column
	for k, v := range m {
		if _, ok := r.mapping[k]; ok {
			return fmt.Errorf("field mapping %s is unbiguous", k)
		}
		r.mapping[k] = v
	}
	// release memory
	r.results = nil
	r.associations = nil
	return nil
}

// ResultTo implements ResultMapper interface.
func (r *resultMapTag) ResultTo(rv reflect.Value, rows *sql.Rows) error {
	if rv.Kind() != reflect.Ptr {
		return errors.New("result must be a pointer")
	}

	// rv must be a pointer to slice or array
	rv = reflect.Indirect(rv)

	if rv.Kind() == reflect.Slice {
		return r.resultToSlice(rv, rows)
	}
	return r.resultToStruct(rv, rows)
}

func (r *resultMapTag) resultToSlice(rv reflect.Value, rows *sql.Rows) error {
	if len(r.collectionGroup) > 0 {
		return errors.New("collection is not supported in slice")
	}
	el := rv.Type().Elem()

	isPtr := el.Kind() == reflect.Ptr

	if isPtr {
		el = el.Elem()
	}

	if el.Kind() != reflect.Struct {
		return errors.New("slice element must be a struct")
	}

	values := reflect.MakeSlice(rv.Type(), 0, 0)

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

		var dest = make([]interface{}, len(columns))

		for index, column := range columns {
			if names, ok := r.mapping[column]; ok {
				var field reflect.Value
				for _, name := range names {
					field = el.FieldByName(name)
					if !field.IsValid() {
						return fmt.Errorf("field %s is not valid", name)
					}
				}
				if !field.IsValid() {
					return fmt.Errorf("field %s is not valid", names)
				}
				dest[index] = field.Addr().Interface()
			} else {
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
			values = reflect.Append(values, rv)
		} else {
			values = reflect.Append(values, el)
		}
	}

	rv.Set(values)

	return nil
}

func (r *resultMapTag) resultToStruct(rv reflect.Value, rows *sql.Rows) error {
	if rv.Kind() != reflect.Struct {
		return errors.New("result must be a struct")
	}
	// check collection group is valid
	if len(r.collectionGroup) > 0 {
		for _, collection := range r.collectionGroup {
			// try to find the field
			field := rv.FieldByName(collection.property)
			if !field.IsValid() {
				return fmt.Errorf("field %s is not valid", collection.property)
			}
			if field.Kind() != reflect.Slice {
				return fmt.Errorf("field %s is not a slice", collection.property)
			}
		}
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	var first = true

	var unFoundColumnsIndex []int
	// now, we start scan rows
	for rows.Next() {
		var dest = make([]interface{}, len(columns))
		for i := range dest {
			dest[i] = new(interface{})
		}
		if first {
			for index, column := range columns {
				if cs, ok := r.mapping[column]; ok {
					var field = rv
					for _, name := range cs {
						field = field.FieldByName(name)
						if !field.IsValid() {
							return fmt.Errorf("field %s is not valid", name)
						}
					}
					if !field.IsValid() {
						return fmt.Errorf("field %s is not valid", cs)
					}
					dest[index] = field.Addr().Interface()
				} else {
					unFoundColumnsIndex = append(unFoundColumnsIndex, index)
				}
			}
			first = false
		}
		var destStruct = make(map[string]reflect.Value)
		for _, index := range unFoundColumnsIndex {
			column := columns[index]
			for _, collection := range r.collectionGroup {
				mapping := collection.mapping
				if err != nil {
					return err
				}
				if cs, ok := mapping[column]; ok {
					value, ok := destStruct[collection.property]
					if !ok {
						ty := rv.FieldByName(collection.property).Type()
						value = reflect.New(ty.Elem()).Elem()
						destStruct[collection.property] = value
					}
					var field = reflect.Indirect(value)
					for _, name := range cs[1:] {
						field = field.FieldByName(name)
						if !field.IsValid() {
							return fmt.Errorf("field %s is not valid", name)
						}
					}
					if !field.IsValid() {
						return fmt.Errorf("field %s is not valid", cs)
					}
					dest[index] = field.Addr().Interface()
				}
			}
		}

		if err := rows.Scan(dest...); err != nil {
			return err
		}

		for column, value := range destStruct {
			field := rv.FieldByName(column)
			field.Set(reflect.Append(field, value))
		}
	}

	return nil
}

// ID returns id of resultMapTag.
func (r *resultMapTag) ID() string {
	return r.id
}

// result defines a result mapping.
type result struct {
	// property is the name of the property to map to.
	property string
	// column is the name of the column to map from.
	column string
}

// resultGroup defines a group of result mappings.
type resultGroup []*result

// mapping returns a mapping of column to property.
func (r resultGroup) mapping() (map[string][]string, error) {
	m := make(map[string][]string)
	for _, v := range r {
		if _, ok := m[v.column]; ok {
			return nil, fmt.Errorf("field mapping %s is unbiguous", v.column)
		}
		m[v.column] = append(m[v.column], v.property)
	}
	return m, nil
}

// association is a collection of results and associations.
type association struct {
	property     string
	results      resultGroup
	associations associationGroup
}

// mapping returns a mapping of column to property.
func (a association) mapping() (map[string][]string, error) {
	m := make(map[string][]string)

	// add results to mapping
	for _, v := range a.results {

		// check if there is any duplicate column
		if _, ok := m[v.column]; ok {
			return nil, fmt.Errorf("field mapping %s is unbiguous", v.column)
		}
		m[v.column] = append(m[v.column], a.property, v.property)
	}

	// add associations to mapping
	for _, v := range a.associations {
		mm, err := v.mapping()
		if err != nil {
			return nil, err
		}

		// check if there is any duplicate column
		for k, v := range mm {
			if _, ok := m[k]; ok {
				return nil, fmt.Errorf("field mapping %s is unbiguous", k)
			}
			m[k] = append(m[k], append([]string{a.property}, v...)...)
		}
	}
	return m, nil
}

// associationGroup defines a group of association mappings.
type associationGroup []*association

// mapping returns a mapping of column to property.
func (a associationGroup) mapping() (map[string][]string, error) {
	m := make(map[string][]string)
	for _, v := range a {
		mm, err := v.mapping()
		if err != nil {
			return nil, err
		}
		for k, v := range mm {
			if _, ok := m[k]; ok {
				return nil, fmt.Errorf("field mapping %s is unbiguous", k)
			}
			m[k] = append(m[k], v...)
		}
	}
	return m, nil
}

type collection struct {
	// property is the name of the property to map to.
	property         string
	resultGroup      resultGroup
	associationGroup associationGroup
	mapping          map[string][]string
}

func (c *collection) init() error {
	c.mapping = make(map[string][]string)
	// add results to mapping
	for _, v := range c.resultGroup {

		// check if there is any duplicate column
		if _, ok := c.mapping[v.column]; ok {
			return fmt.Errorf("field mapping %s is unbiguous", v.column)
		}
		c.mapping[v.column] = append(c.mapping[v.column], c.property, v.property)
	}

	// add associations to mapping
	for _, v := range c.associationGroup {
		mm, err := v.mapping()
		if err != nil {
			return err
		}

		// check if there is any duplicate column
		for k, v := range mm {
			if _, ok := c.mapping[k]; ok {
				return fmt.Errorf("field mapping %s is unbiguous", k)
			}
			c.mapping[k] = append(c.mapping[k], append([]string{c.property}, v...)...)
		}
	}
	return nil
}

type collectionGroup []*collection
