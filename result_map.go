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

// ResultTo implements ResultMapper interface.
func (r *resultMapNode) ResultTo(rv reflect.Value, rows *sql.Rows) error {
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

// resultToSlice scans rows to slice with given entity
func (r *resultMapNode) resultToSlice(rv reflect.Value, rows *sql.Rows) error {
	// get the element type of slice or array
	el := rv.Type().Elem()

	// if it's a pointer, get the element type of pointer
	isPtr := el.Kind() == reflect.Ptr

	if isPtr {
		el = el.Elem()
	}

	// check if it's a struct
	if el.Kind() != reflect.Struct {
		return errors.New("slice element must be a struct")
	}

	// result slice of this entity
	values := reflect.MakeSlice(rv.Type(), 0, 0)

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	// checked is a flag to check if dest has sql.RawBytes and only check once
	var checked bool

	var indexes = make(map[string][]int)

	var unfoundedIndex = make(map[string]int)

	// now, we start scan rows
	for rows.Next() {

		// make a new element of slice
		elValue := reflect.New(el).Elem()

		tp := elValue.Type()

		// dest is a slice of pointers to fields of element
		var dest = make([]interface{}, len(columns))

		// initialize dest with default pointer insures that all fields can be scanned
		for i := range dest {
			dest[i] = new(interface{})
		}

		// start do some magic
		if !checked {
			// try to find the field of element by column name
			for index, column := range columns {
				// if we have a mapping for this column, use it
				if cs, ok := r.mapping[column]; ok {
					var field reflect.StructField
					indexes[column] = make([]int, 0, len(cs))
					// try to find the field of element by column name
					for i, name := range cs {
						var ok bool
						// if it is first time to find this field, we need to find it
						if i == 0 {
							field, ok = tp.FieldByName(name)
						} else {
							// if it is not first time to find this field, we need to find it from the last field
							field, ok = field.Type.FieldByName(name)
						}
						// if we can't find the field, return error
						if !ok {
							return fmt.Errorf("field %s is not valid", name)
						}
						// append the index of field to indexes
						indexes[column] = append(indexes[column], field.Index[0])
					}
				} else {
					// if we don't have a mapping for this column, set it to unfounded Index
					unfoundedIndex[column] = index
				}
			}

			// try to find those fields which are not found in mapping
			if r.HasCollection() {
				for _, index := range unfoundedIndex {
					column := columns[index]
					// find form collection
					for _, coll := range r.collectionGroup {
						mapping := coll.mapping
						// if we have a mapping for this column, use it
						if names, ok := mapping[column]; ok {
							// try to find the destination field index of element by column name
							indexes[column] = make([]int, 0, len(names))

							var field reflect.StructField
							for i, name := range names {
								var ok bool
								if i == 0 {
									field, ok = tp.FieldByName(name)
								} else {
									if field.Type.Kind() == reflect.Slice {
										el := field.Type.Elem()
										if el.Kind() == reflect.Ptr {
											el = el.Elem()
										}
										field, ok = el.FieldByName(name)
									}
								}
								if !ok {
									return fmt.Errorf("field %s is not valid", name)
								}
								indexes[column] = append(indexes[column], field.Index[0])
							}
						}
					}
				}
			}
		}

		var collectionMapping map[string]collectionIndexItem

		for column, _ := range unfoundedIndex {
			for _, collection := range r.collectionGroup {
				mapping := collection.mapping
				if _, ok := mapping[column]; ok {
					if collectionMapping == nil {
						collectionMapping = make(map[string]collectionIndexItem)
					}
					_, ok := collectionMapping[collection.property]
					if !ok {
						// slice must be a slice, we have checked it before
						slice := elValue.FieldByName(collection.property).Type()
						// slice element must be a struct, we have checked it before
						elType := slice.Elem()
						// if it's a pointer, get the element type of pointer
						tyIsPtr := elType.Kind() == reflect.Ptr
						if tyIsPtr {
							elType = elType.Elem()
						}
						collectionMapping[collection.property] = collectionIndexItem{
							column: column,
							rv:     reflect.New(elType).Elem(),
							isPtr:  tyIsPtr,
						}
					}
				}
			}
		}

		for index, column := range columns {
			var field = elValue
			var start int
			for _, value := range collectionMapping {
				if value.column == column {
					field = value.rv
					start = 1
					break
				}
			}
			if cs, ok := indexes[column]; ok {
				for _, i := range cs[start:] {
					field = field.Field(i)
				}
				dest[index] = field.Addr().Interface()
			}
		}

		// from now on, we have got all the fields of element

		// check if the dest has sql.RawBytes
		if !checked {
			for _, dp := range dest {
				if _, ok := dp.(*sql.RawBytes); ok {
					return errors.New("sql: RawBytes isn't allowed on Row.Scan")
				}
			}
			checked = true
		}

		if err := rows.Scan(dest...); err != nil {
			return err
		}

		if values.Len() == 0 {
			if isPtr {
				elValue = elValue.Addr()
			}
			values = reflect.Append(values, elValue)
		}
		var isNew = true
		if r.HasPk() && r.HasCollection() {
			for i := 0; i < values.Len(); i++ {
				loopValue := values.Index(i)
				if loopValue.FieldByName(r.pk.property).Interface() == elValue.FieldByName(r.pk.property).Interface() {
					isNew = false
					for fieldName, value := range collectionMapping {
						field := loopValue.FieldByName(fieldName)
						x := value.rv
						if value.isPtr {
							x = value.rv.Addr()
						}
						field.Set(reflect.Append(field, x))
					}
					break
				}
			}
		}
		if isNew {
			if isPtr {
				elValue = elValue.Addr()
			}
			values = reflect.Append(values, elValue)
		}
	}

	rv.Set(values)

	return nil
}

func (r *resultMapNode) resultToStruct(rv reflect.Value, rows *sql.Rows) error {
	if rv.Kind() != reflect.Struct {
		return errors.New("slice element must be a struct")
	}

	// check collection is valid
	if r.HasCollection() {
		for _, coll := range r.collectionGroup {
			field := rv.FieldByName(coll.property)
			if field.Kind() != reflect.Slice {
				return errors.New("collection field must be a slice")
			}
			el := field.Type().Elem()
			if el.Kind() == reflect.Ptr {
				el = el.Elem()
			}
			if el.Kind() != reflect.Struct {
				return errors.New("collection field must be a slice of struct")
			}
		}
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	var checked bool

	var indexes = make(map[string][]int)

	var unFoundedIndex = make(map[string]int)

	tp := rv.Type()

	var pk interface{}

	// now, we start scan rows
	for rows.Next() {

		if checked && !r.HasCollection() {
			return errors.New("result has more than one row but no collection")
		}

		var elValue = rv

		if checked {
			elValue = reflect.New(tp).Elem()
		}

		var dest = make([]interface{}, len(columns))
		for i := range dest {
			dest[i] = new(interface{})
		}

		if !checked {
			for index, column := range columns {
				if cs, ok := r.mapping[column]; ok {
					var field reflect.StructField
					indexes[column] = make([]int, 0, len(cs))
					for i, name := range cs {
						var ok bool
						if i == 0 {
							field, ok = tp.FieldByName(name)
						} else {
							field, ok = field.Type.FieldByName(name)
						}
						if !ok {
							return fmt.Errorf("field %s is not valid", name)
						}
						indexes[column] = append(indexes[column], field.Index[0])
					}
				} else {
					unFoundedIndex[column] = index
				}
			}

			if r.HasCollection() {
				for _, index := range unFoundedIndex {
					column := columns[index]
					for _, collection := range r.collectionGroup {
						mapping := collection.mapping
						if cs, ok := mapping[column]; ok {
							indexes[column] = make([]int, 0, len(cs))
							var field reflect.StructField
							for i, name := range cs {
								var ok bool
								if i == 0 {
									field, ok = tp.FieldByName(name)
								} else {
									if field.Type.Kind() == reflect.Slice {
										el := field.Type.Elem()
										if el.Kind() == reflect.Ptr {
											el = el.Elem()
										}
										field, ok = el.FieldByName(name)
									}
								}
								if !ok {
									return fmt.Errorf("field %s is not valid", name)
								}
								indexes[column] = append(indexes[column], field.Index[0])
							}
						}
					}
				}
			}
		}

		var collectionMapping map[string]collectionIndexItem

		for column, _ := range unFoundedIndex {
			for _, collection := range r.collectionGroup {
				mapping := collection.mapping
				if _, ok := mapping[column]; ok {
					if collectionMapping == nil {
						collectionMapping = make(map[string]collectionIndexItem, 0)
					}
					_, ok := collectionMapping[collection.property]
					if !ok {
						ty := elValue.FieldByName(collection.property).Type()
						elType := ty.Elem()
						isPtr := elType.Kind() == reflect.Ptr
						if isPtr {
							elType = elType.Elem()
						}
						value := reflect.New(elType).Elem()
						collectionMapping[collection.property] = collectionIndexItem{
							rv:     value,
							column: column,
							isPtr:  isPtr,
						}
					}
				}
			}
		}

		for index, column := range columns {
			var field = elValue
			var start int
			for _, value := range collectionMapping {
				if value.column == column {
					field = value.rv
					start = 1
					break
				}
			}
			if cs, ok := indexes[column]; ok {
				for _, i := range cs[start:] {
					field = field.Field(i)
				}
				dest[index] = field.Addr().Interface()
			}
		}

		if !checked {
			for _, dp := range dest {
				if _, ok := dp.(*sql.RawBytes); ok {
					return errors.New("sql: RawBytes isn't allowed on Row.Scan")
				}
			}
			checked = true
		}

		if err := rows.Scan(dest...); err != nil {
			return err
		}

		if r.HasPk() && pk == nil {
			pk = elValue.FieldByName(r.pk.property).Interface()
		}

		if r.HasPk() && r.HasCollection() {
			currentPkValue := elValue.FieldByName(r.pk.property).Interface()
			if pk != currentPkValue {
				return errors.New("result has more than one row but no collection")
			}
			for _, collection := range r.collectionGroup {
				value := rv.FieldByName(collection.property)
				item := collectionMapping[collection.property]
				field := item.rv
				if item.isPtr {
					field = field.Addr()
				}
				value.Set(reflect.Append(value, field))
			}
		}
	}

	return nil
}

type collectionIndexItem struct {
	isPtr  bool
	column string
	rv     reflect.Value
}
