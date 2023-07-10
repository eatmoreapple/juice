/*
Copyright 2023 eatmoreapple

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

// ErrTooManyRows is returned when the result set has too many rows but excepted only one row.
var ErrTooManyRows = errors.New("juice: too many rows in result set")

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

	// if it has any row data
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	var cd ColumnDestination = &rowDestination{}

	dest, err := cd.Destination(rv, columns)
	if err != nil {
		return err
	}
	// scan the row data to dest
	if err = rows.Scan(dest...); err != nil {
		return err
	}
	if err = rows.Err(); err != nil {
		return err
	}
	// return ErrTooManyRows if there are more than one row data
	// it means the result is a slice, but the destination is not
	if rows.Next() {
		return ErrTooManyRows
	}
	return nil
}

// RowsResultMap is a ResultMap that maps a rowDestination to a slice type.
type RowsResultMap struct{}

// ResultTo implements ResultMapper interface.
func (RowsResultMap) ResultTo(rv reflect.Value, rows *sql.Rows) error {

	if rv.Kind() != reflect.Ptr {
		return errors.New(" must be a pointer")
	}

	// rv must be a pointer to slice or array
	rv = rv.Elem()

	// get the element type of slice or array
	el := rv.Type().Elem()

	// if it's a pointer, get the element type of pointer
	isPtr := el.Kind() == reflect.Ptr

	// make a new slice of element type
	ret := reflect.MakeSlice(rv.Type(), 0, 0)

	// get the element type of pointer
	el = kindIndirect(el)

	// get columns from rows
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	var cd ColumnDestination = &rowDestination{}
	// if it's a struct, scan rows to struct
	// now, we start scan rows
	values := make([]reflect.Value, 0)

	for rows.Next() {

		// make a new element of slice
		nrv := reflect.New(el)

		// get the Value of element
		nel := nrv.Elem()

		// get the destination of element

		dest, err := cd.Destination(nel, columns)

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

		// append the element to
		if isPtr {
			values = append(values, nrv)
		} else {
			values = append(values, nel)
		}
	}

	// set result to given entity
	rv.Set(reflect.Append(ret, values...))

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
		if err = rows.Scan(dest...); err != nil {
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

	var pk any

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
		if err = rows.Scan(dest...); err != nil {
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
			// set group
			for _, group := range r.collectionGroup {
				value := rv.FieldByName(group.property)
				item := cd.collectionMapping[group.property]
				field := item.rv
				if item.isPtr {
					field = field.Addr()
				}
				value.Set(reflect.Append(value, field))
			}
		}

		checked = true
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

// discardIndex is the index of the discard column which will be ignored.
const discardIndex = -1

// ColumnDestination is a column destination which can be used to scan a row.
type ColumnDestination interface {
	// Destination returns the destination for the given reflect value and column.
	Destination(rv reflect.Value, column []string) ([]any, error)
}

// rowDestination is a ColumnDestination which can be used to scan a row.
type rowDestination struct {
	// indexes stores the index of the column in the struct.
	// if the index is discardIndex, the column will be ignored.
	// this could not be used to for deep struct scan.
	indexes []int

	// checked is a flag to check if the dest has sql.RawBytes
	checked bool
}

// Destination returns the destination for the given reflect value and column.
func (s *rowDestination) Destination(rv reflect.Value, columns []string) ([]any, error) {
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

func (s *rowDestination) destination(rv reflect.Value, columns []string) ([]any, error) {
	// if there is only one column, we can use the value directly.
	if len(columns) == 1 {
		return []any{rv.Addr().Interface()}, nil
	}
	if rv.Kind() == reflect.Struct {
		return s.destinationForStruct(rv, columns)
	}
	return nil, fmt.Errorf("expected struct, but got %s", rv.Type())
}

func (s *rowDestination) destinationForStruct(rv reflect.Value, columns []string) ([]any, error) {
	if len(s.indexes) == 0 {
		s.setIndexes(rv, columns)
	}
	dest := make([]any, len(columns))
	for i, index := range s.indexes {
		if index == discardIndex {
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
		s.indexes[i] = discardIndex
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
// I have no idea what i am doing
type resultMapColumnDestination struct {
	resultMap         *resultMapNode
	indexes           map[string][]int
	unFoundedIndex    map[string]int
	collectionMapping collectionItemMapping
	checked           bool
}

// Destination returns the destination for the given reflect value and column.
func (s *resultMapColumnDestination) Destination(rv reflect.Value, columns []string) ([]any, error) {
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

func (s *resultMapColumnDestination) destination(rv reflect.Value, columns []string) ([]any, error) {
	dest := make([]any, len(columns))
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
				var found bool
				// if it is first time to find this field, we need to find it
				if i == 0 {
					field, found = tp.FieldByName(name)
				} else {
					// if it is not first time to find this field, we need to find it from the last field
					field, found = field.Type.FieldByName(name)
				}
				// if we can't find the field, return error
				if !found {
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
			// find form group
			column := columns[index]
			for _, group := range s.resultMap.collectionGroup {
				mapping := group.mapping
				// if we have a mapping for this column, use it
				if names, ok := mapping[column]; ok {
					// try to find the destination field index of element by column name
					s.indexes[column] = make([]int, 0, len(names))
					var field reflect.StructField
					for i, name := range names {
						var found bool
						if i == 0 {
							field, found = tp.FieldByName(name)
						} else {
							if field.Type.Kind() == reflect.Slice {
								el := field.Type.Elem()
								if el.Kind() == reflect.Ptr {
									el = el.Elem()
								}
								field, found = el.FieldByName(name)
							}
						}
						if !found {
							return fmt.Errorf("field %s is not valid", name)
						}
						s.indexes[column] = append(s.indexes[column], field.Index[0])
					}
				}
			}
		}
	}

	var elValue = rv

	for column := range s.unFoundedIndex {
		for _, group := range s.resultMap.collectionGroup {
			mapping := group.mapping
			if _, ok := mapping[column]; ok {
				if s.collectionMapping == nil {
					s.collectionMapping = make(collectionItemMapping, 0)
				}
				_, ok = s.collectionMapping[group.property]
				if !ok {
					// sliceType must be a slice, we have checked it before
					sliceType := elValue.FieldByName(group.property).Type()
					// sliceType element must be a struct, we have checked it before
					elType := sliceType.Elem()
					// if it's a pointer, get the element type of pointer
					tyIsPtr := elType.Kind() == reflect.Ptr
					if tyIsPtr {
						elType = elType.Elem()
					}
					value := reflect.New(elType).Elem()
					s.collectionMapping[group.property] = &collectionItem{
						rv:      value,
						columns: make(map[string]struct{}),
						isPtr:   tyIsPtr,
					}
				}
				s.collectionMapping[group.property].columns[column] = struct{}{}
				break
			}
		}
	}
	return nil
}

func checkDestination(dest []any) error {
	for _, dp := range dest {
		if _, ok := dp.(*sql.RawBytes); ok {
			return errors.New("sql: RawBytes isn't allowed on scan")
		}
	}
	return nil
}
