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

// RowResultMap is a ResultMap that maps a rowDestination to a non-slice type.
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

	var cd ColumnDestination = &rowDestination{}

	dest, err := cd.Destination(rv, columns)
	if err != nil {
		return err
	}
	if err = rows.Scan(dest...); err != nil {
		return err
	}
	return rows.Err()
}

// RowsResultMap is a ResultMap that maps a rowDestination to a slice type.
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
	// now, we start scan rows
	for rows.Next() {

		// make a new element of slice
		rv := reflect.New(el)

		// get the Value of element
		el := rv.Elem()

		// get the destination of element
		var cd ColumnDestination = &rowDestination{}

		dest, err := cd.Destination(el, columns)

		if err != nil {
			return err
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

	var cd = &resultMapColumnDestination{resultMap: r}

	// now, we start scan rows
	for rows.Next() {

		// make a new element of slice
		elValue := reflect.New(el).Elem()

		dest, err := cd.Destination(elValue, columns)

		if err != nil {
			return err
		}

		// reset index from collection
		for index, column := range columns {
			var field = elValue
			var start int
			if item, ok := cd.collectionMapping.getCollectionIndexItem(column); ok {
				field = item.rv
				start = 1
			}
			if cs, ok := cd.indexes[column]; ok {
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

		// scan the rowDestination with dest
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

				// if it's a pointer, get the element
				if loopValue.Kind() == reflect.Ptr {
					loopValue = loopValue.Elem()
				}

				// get primary key from the loop element
				loopPk := loopValue.FieldByName(r.pk.property).Interface()

				// get primary key from the current
				currentPk := elValue.FieldByName(r.pk.property).Interface()

				if currentPk == loopPk {
					isNew = false
					// set the value of collection
					cd.collectionMapping.setCollection(loopValue)
					break
				}
			}
		}

		// if the element is new, append it to collection
		if isNew {
			cd.collectionMapping.setCollection(elValue)
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

	tp := rv.Type()

	var pk interface{}

	var cd = &resultMapColumnDestination{resultMap: r}

	// now, we start scan rows
	for rows.Next() {

		// not first time, but has more than one rows
		if checked && !r.HasCollection() {
			return errors.New("result has more than one rowDestination but no collection")
		}

		var elValue = rv

		// if not first time, we need to build a new instance
		if checked {
			elValue = reflect.New(tp).Elem()
		}

		dest, err := cd.Destination(elValue, columns)

		if err != nil {
			return err
		}

		// scan the rowDestination with dest
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
				return errors.New("result has more than one rowDestination but no collection")
			}
			// set collection
			for _, collection := range r.collectionGroup {
				value := rv.FieldByName(collection.property)
				item := cd.collectionMapping[collection.property]
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

// ColumnDestination is a column destination which can be used to scan a row.
type ColumnDestination interface {
	// Destination returns the destination for the given reflect value and column.
	Destination(rv reflect.Value, column []string) ([]interface{}, error)
}

type rowDestination struct {
	indexes []int
	checked bool
}

// Destination returns the destination for the given reflect value and column.
func (s *rowDestination) Destination(rv reflect.Value, columns []string) ([]interface{}, error) {
	dest, err := s.destination(rv, columns)
	if err != nil {
		return nil, err
	}
	if !s.checked {
		if err = checkDestination(dest); err != nil {
			return nil, err
		}
		s.checked = true
	}
	return dest, nil
}

func (s *rowDestination) destination(rv reflect.Value, columns []string) ([]interface{}, error) {
	if rv.Kind() == reflect.Struct {
		return s.destinationForStruct(rv, columns)
	}
	if len(columns) != 1 {
		return nil, errors.New("only one column is allowed for non-struct")
	}
	return []interface{}{rv.Addr().Interface()}, nil
}

func (s *rowDestination) destinationForStruct(rv reflect.Value, columns []string) ([]interface{}, error) {
	if len(s.indexes) == 0 {
		s.setIndexes(rv, columns)
	}
	dest := make([]interface{}, len(columns))
	for i, index := range s.indexes {
		if index == -1 {
			dest[i] = new(any)
		} else {
			dest[i] = rv.Field(index).Addr().Interface()
		}
	}
	return dest, nil
}

func (s *rowDestination) setIndexes(rv reflect.Value, columns []string) {
	tp := rv.Type()
	s.indexes = make([]int, len(columns))

	// index is the index of the field in the struct
	for i := range s.indexes {
		s.indexes[i] = -1
	}
	for i := 0; i < tp.NumField(); i++ {
		field := tp.Field(i)
		tag := field.Tag.Get("column")
		if tag == "" || tag == "-" {
			continue
		}
		for index, column := range columns {
			if tag == column {
				s.indexes[index] = i
				break
			}
		}
	}
}

// TODO fixme
type resultMapColumnDestination struct {
	resultMap         *resultMapNode
	indexes           map[string][]int
	unFoundedIndex    map[string]int
	collectionMapping collectionItemMapping
	checked           bool
}

// Destination returns the destination for the given reflect value and column.
func (s *resultMapColumnDestination) Destination(rv reflect.Value, columns []string) ([]interface{}, error) {
	dest, err := s.destination(rv, columns)
	if err != nil {
		return nil, err
	}
	if !s.checked {
		if err = checkDestination(dest); err != nil {
			return nil, err
		}
		s.checked = true
	}
	return dest, nil
}

func (s *resultMapColumnDestination) destination(rv reflect.Value, columns []string) ([]interface{}, error) {
	dest := make([]interface{}, len(columns))
	if len(s.indexes) == 0 {
		if err := s.setIndexes(rv, columns); err != nil {
			return nil, err
		}
	}
	for i := range dest {
		dest[i] = new(any)
	}
	for index, column := range columns {
		var field = rv
		var start int
		// try to find it form the collection
		if item, ok := s.collectionMapping.getCollectionIndexItem(column); ok {
			field = item.rv
			start = 1
		}
		if cs, ok := s.indexes[column]; ok {
			for _, i := range cs[start:] {
				field = field.Field(i)
				// does not support pointer
				// it will make the code more complex and slower
				if field.Kind() == reflect.Ptr {
					return nil, errors.New("struct field must not be a pointer")
				}
			}
			dest[index] = field.Addr().Interface()
		}
	}
	return dest, nil
}

func (s *resultMapColumnDestination) setIndexes(rv reflect.Value, columns []string) error {
	s.indexes = make(map[string][]int)

	s.unFoundedIndex = make(map[string]int)

	tp := rv.Type()

	for index, column := range columns {
		// if we have a mapping for this column, use it
		if cs, ok := s.resultMap.mapping[column]; ok {
			var field reflect.StructField
			s.indexes[column] = make([]int, 0, len(cs))
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
				s.indexes[column] = append(s.indexes[column], field.Index...)
			}
		} else {
			// if we don't have a mapping for this column, set it to unfounded Index
			s.unFoundedIndex[column] = index
		}
	}

	// try to find those fields which are not found in mapping
	if s.resultMap.HasCollection() {
		for _, index := range s.unFoundedIndex {
			// find form collection
			column := columns[index]
			for _, collection := range s.resultMap.collectionGroup {
				mapping := collection.mapping
				// if we have a mapping for this column, use it
				if names, ok := mapping[column]; ok {
					// try to find the destination field index of element by column name
					s.indexes[column] = make([]int, 0, len(names))
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
						s.indexes[column] = append(s.indexes[column], field.Index[0])
					}
				}
			}
		}
	}

	var elValue = rv

	for column, _ := range s.unFoundedIndex {
		for _, collection := range s.resultMap.collectionGroup {
			mapping := collection.mapping
			if _, ok := mapping[column]; ok {
				if s.collectionMapping == nil {
					s.collectionMapping = make(collectionItemMapping, 0)
				}
				_, ok := s.collectionMapping[collection.property]
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
					s.collectionMapping[collection.property] = &collectionItem{
						rv:      value,
						columns: make(map[string]struct{}),
						isPtr:   tyIsPtr,
					}
				}
				s.collectionMapping[collection.property].columns[column] = struct{}{}
				break
			}
		}
	}
	return nil
}

func checkDestination(dest []interface{}) error {
	for _, dp := range dest {
		if _, ok := dp.(*sql.RawBytes); ok {
			return errors.New("sql: RawBytes isn't allowed on scan")
		}
	}
	return nil
}
