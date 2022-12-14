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

	// check collection is valid
	if r.HasCollection() {
		// does not found id tag for current tag
		if !r.HasPk() {
			return errors.New("collection must have a primary key")
		}
		for _, coll := range r.collectionGroup {
			field, ok := el.FieldByName(coll.property)
			if !ok {
				return fmt.Errorf("collection property %s not found", coll.property)
			}
			if field.Type.Kind() != reflect.Slice {
				return errors.New("collection field must be a slice")
			}
			// get slice element type
			el := kindIndirect(field.Type.Elem())

			// collection element must be a struct
			if el.Kind() != reflect.Struct {
				return errors.New("collection field must be a slice of struct")
			}
		}
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
						indexes[column] = append(indexes[column], field.Index...)
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

		var collectionMapping collectionItemMapping

		for column, _ := range unfoundedIndex {
			for _, collection := range r.collectionGroup {
				mapping := collection.mapping
				if _, ok := mapping[column]; ok {
					if collectionMapping == nil {
						collectionMapping = make(map[string]*collectionItem)
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
						collectionMapping[collection.property] = &collectionItem{
							rv:      reflect.New(elType).Elem(),
							isPtr:   tyIsPtr,
							columns: make(map[string]struct{}),
						}
					}
					collectionMapping[collection.property].columns[column] = struct{}{}
					break
				}
			}
		}

		// reset index from collection
		for index, column := range columns {
			var field = elValue
			var start int
			if item, ok := collectionMapping.getCollectionIndexItem(column); ok {
				field = item.rv
				start = 1
			}
			if cs, ok := indexes[column]; ok {
				for _, i := range cs[start:] {
					field = field.Field(i)
					// does not support pointer
					// I don't want to support pointer, it will make the code more complex and slower
					if field.Kind() == reflect.Ptr {
						return errors.New("struct field must not be a pointer")
					}
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

		// scan the row with dest
		if err := rows.Scan(dest...); err != nil {
			return err
		}

		// here we have got all the fields of element, but we still need to set the value of collection
		var isNew = true
		if r.HasCollection() {

			// start a loop to find the element in collection
			// if the primary key of loop element is equal to the primary key of current, we have found it
			for i := 0; i < values.Len(); i++ {
				loopValue := values.Index(i)

				// get primary key from the loop element
				loopPk := loopValue.FieldByName(r.pk.property).Interface()

				// get primary key from the current
				currentPk := elValue.FieldByName(r.pk.property).Interface()

				if currentPk == loopPk {
					isNew = false
					// set the value of collection
					collectionMapping.setCollection(loopValue)
					break
				}
			}
		}

		// if the element is new, append it to collection
		if isNew {
			collectionMapping.setCollection(elValue)
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
		// does not found id tag for current tag
		if !r.HasPk() {
			return errors.New("collection must have a primary key")
		}
		for _, coll := range r.collectionGroup {
			field := rv.FieldByName(coll.property)

			// collection must be a slice
			if field.Kind() != reflect.Slice {
				return errors.New("collection field must be a slice")
			}

			// get slice element type
			el := kindIndirect(field.Type().Elem())

			// collection element must be a struct
			if el.Kind() != reflect.Struct {
				return errors.New("collection field must be a slice of struct")
			}
		}
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	// checked is a flag to check if the dest has sql.RawBytes
	var checked bool

	var indexes = make(map[string][]int)

	var unFoundedIndex = make(map[string]int)

	tp := rv.Type()

	var pk interface{}

	// now, we start scan rows
	for rows.Next() {

		// not first time, but has more than one rows
		if checked && !r.HasCollection() {
			return errors.New("result has more than one row but no collection")
		}

		var elValue = rv

		// if not first time, we need to build a new instance
		if checked {
			elValue = reflect.New(tp).Elem()
		}

		// dest is a slice of pointers to fields of element
		var dest = make([]interface{}, len(columns))

		// initialize dest with default pointer insures that all fields can be scanned
		for i := range dest {
			dest[i] = new(interface{})
		}

		// start to initialize
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
						indexes[column] = append(indexes[column], field.Index...)
					}
				} else {
					// if we don't have a mapping for this column, set it to unfounded Index
					unFoundedIndex[column] = index
				}
			}

			// try to find those fields which are not found in mapping
			if r.HasCollection() {
				for _, index := range unFoundedIndex {
					// find form collection
					column := columns[index]
					for _, collection := range r.collectionGroup {
						mapping := collection.mapping
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

		var collectionMapping collectionItemMapping

		for column, _ := range unFoundedIndex {
			for _, collection := range r.collectionGroup {
				mapping := collection.mapping
				if _, ok := mapping[column]; ok {
					if collectionMapping == nil {
						collectionMapping = make(collectionItemMapping, 0)
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
						value := reflect.New(elType).Elem()
						collectionMapping[collection.property] = &collectionItem{
							rv:      value,
							columns: make(map[string]struct{}),
							isPtr:   tyIsPtr,
						}
					}
					collectionMapping[collection.property].columns[column] = struct{}{}
					break
				}
			}
		}

		// reset index from collection
		for index, column := range columns {
			var field = elValue
			var start int
			// try to find it form the collection
			if item, ok := collectionMapping.getCollectionIndexItem(column); ok {
				field = item.rv
				start = 1
			}
			if cs, ok := indexes[column]; ok {
				for _, i := range cs[start:] {
					field = field.Field(i)
					// does not support pointer
					// it will make the code more complex and slower
					if field.Kind() == reflect.Ptr {
						return errors.New("struct field must not be a pointer")
					}
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

		// scan the row with dest
		if err := rows.Scan(dest...); err != nil {
			return err
		}

		// try to set pk
		if r.HasPk() && pk == nil {
			pk = elValue.FieldByName(r.pk.property).Interface()
		}

		// try to set collection
		if r.HasCollection() {
			currentPk := elValue.FieldByName(r.pk.property).Interface()
			// if the record is correct?
			if pk != currentPk {
				return errors.New("result has more than one row but no collection")
			}
			// set collection
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

// collectionItem is a collection item
type collectionItem struct {
	// isPtr if the collection item is a pointer
	isPtr bool
	// columns is the columns of the collection item
	columns map[string]struct{}
	// rv is the new reflect value of the collection item
	rv reflect.Value
}

// collectionItemMapping is a collection item mapping
type collectionItemMapping map[string]*collectionItem

// getCollectionIndexItem returns the collectionItem for the given column.
func (c collectionItemMapping) getCollectionIndexItem(column string) (*collectionItem, bool) {
	for _, value := range c {
		if _, ok := value.columns[column]; ok {
			return value, true
		}
	}
	return nil, false
}

// setCollection sets the collectionItem for the given value.
func (c collectionItemMapping) setCollection(rv reflect.Value) {
	for fieldName, value := range c {
		field := rv.FieldByName(fieldName)
		x := value.rv
		if value.isPtr {
			x = value.rv.Addr()
		}
		field.Set(reflect.Append(field, x))
	}
}
