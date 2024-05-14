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

// Package juice provides a set of utilities for mapping database query results to Go data structures.
package juice

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
)

// ErrTooManyRows is returned when the result set has too many rows but excepted only one row.
var ErrTooManyRows = errors.New("juice: too many rows in result set")

// ResultMap is an interface that defines a method for mapping database query results to Go data structures.
type ResultMap interface {
	// MapTo maps the data from the SQL row to the provided reflect.Value.
	MapTo(rv reflect.Value, row *sql.Rows) error
}

// SingleRowResultMap is a ResultMap that maps a rowDestination to a non-slice type.
type SingleRowResultMap struct{}

// MapTo implements ResultMapper interface.
// It maps the data from the SQL row to the provided reflect.Value.
// If more than one row is returned from the query, it returns an ErrTooManyRows error.
func (SingleRowResultMap) MapTo(rv reflect.Value, rows *sql.Rows) error {
	if rv.Kind() != reflect.Ptr {
		return ErrPointerRequired
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

	rv = reflect.Indirect(rv)

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

// MultiRowsResultMap is a ResultMap that maps a rowDestination to a slice type.
type MultiRowsResultMap struct{}

// MapTo implements ResultMapper interface.
// It maps the data from the SQL row to the provided reflect.Value.
// It maps each row to a new element in a slice.
func (MultiRowsResultMap) MapTo(rv reflect.Value, rows *sql.Rows) error {

	if rv.Kind() != reflect.Ptr {
		return ErrPointerRequired
	}

	// rv must be a pointer to slice or array
	rv = rv.Elem()

	// get the element type of slice or array
	el := rv.Type().Elem()

	// if it's a pointer, get the element type of pointer
	isPtr := el.Kind() == reflect.Ptr

	// get the element type of pointer
	if el.Kind() == reflect.Ptr {
		el = el.Elem()
	}

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

		// append the element to
		if isPtr {
			values = append(values, nrv)
		} else {
			values = append(values, nel)
		}
	}

	// has error, return it
	if err = rows.Err(); err != nil {
		return err
	}

	// make a new slice of element type
	ret := reflect.MakeSlice(rv.Type(), 0, len(values))

	// set result to given entity
	rv.Set(reflect.Append(ret, values...))

	return nil
}

// MapTo implements ResultMapper interface.
func (r *resultMapNode) MapTo(rv reflect.Value, rows *sql.Rows) error {
	if rv.Kind() != reflect.Ptr {
		return ErrPointerRequired
	}
	rv = reflect.Indirect(rv)

	if rv.Kind() == reflect.Slice {
		return r.resultToSlice(rv, rows)
	}
	return r.resultToStruct(rv, rows)
}

func (r *resultMapNode) binderToStruct(rv reflect.Value, columns []string, rows *sql.Rows) error {
	items, err := r.binders.BindTo(rv.Addr())
	if err != nil {
		return err
	}
	dest := make([]any, len(columns))

	for index, column := range columns {
		addr, exists := items[column]
		if exists {
			dest[index] = addr
		} else {
			// if there is no column in the result, we can set a new instance to it.
			// just discard
			dest[index] = new(any)
		}
	}

	if err = checkDestination(dest); err != nil {
		return err
	}
	return rows.Scan(dest...)
}

// resultToStruct scans rows to struct with given entity
// it's used when the relation is one to one or one to many
func (r *resultMapNode) resultToStruct(rv reflect.Value, rows *sql.Rows) error {
	if rv.Kind() != reflect.Struct {
		return errors.New("slice element must be a struct")
	}
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	for rows.Next() {
		if err = r.binderToStruct(rv, columns, rows); err != nil {
			return err
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return nil
}

func (r *resultMapNode) getValuesFromRows(rows *sql.Rows, el reflect.Type, isPtr bool) ([]reflect.Value, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	values := make([]reflect.Value, 0)
	for rows.Next() {
		instance := reflect.New(el)
		value := instance.Elem()
		if err = r.binderToStruct(value, columns, rows); err != nil {
			return nil, err
		}
		if !isPtr {
			instance = value
		}
		values = append(values, instance)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func (r *resultMapNode) appendValuesWithPrimaryKey(ret reflect.Value, values []reflect.Value) reflect.Value {
	for _, value := range values {
		directValue := reflect.Indirect(value)
		pk := reflect.Indirect(directValue.FieldByName(r.pk.property)).Interface()
		var found bool
		for i := 0; i < ret.Len(); i++ {
			current := reflect.Indirect(ret.Index(i))
			field := current.FieldByName(r.pk.property)
			if found = reflect.Indirect(field).Interface() == pk; found {
				for _, item := range r.collectionGroup {
					loopField := current.FieldByName(item.property)
					loop := loopField
					loop = reflect.AppendSlice(loop, directValue.FieldByName(item.property))
					loopField.Set(loop)
				}
				break
			}
		}
		if !found {
			ret = reflect.Append(ret, value)
		}
	}
	return ret
}

// resultToSlice scans rows to slice with given entity
func (r *resultMapNode) resultToSlice(rv reflect.Value, rows *sql.Rows) error {
	if rv.Kind() != reflect.Slice {
		return errors.New("slice element must be a struct")
	}
	el := rv.Type().Elem()
	isPtr := el.Kind() == reflect.Ptr
	if isPtr {
		el = el.Elem()
	}
	if el.Kind() != reflect.Struct {
		return errors.New("slice element must be a struct")
	}
	if r.pk != nil {
		// check if the primary key is in the columns
		pk, ok := el.FieldByName(r.pk.property)
		if !ok {
			return fmt.Errorf("property %s not found", r.pk.property)
		}
		if !pk.Type.Comparable() {
			return fmt.Errorf("property %s must be comparable", r.pk.property)
		}
	}
	values, err := r.getValuesFromRows(rows, el, isPtr)
	if err != nil {
		return err
	}
	ret := reflect.MakeSlice(rv.Type(), 0, len(values))
	if r.pk != nil {
		ret = r.appendValuesWithPrimaryKey(ret, values)
	} else {
		ret = reflect.Append(ret, values...)
	}
	rv.Set(ret)
	return nil
}

// ColumnDestination is a column destination which can be used to scan a row.
type ColumnDestination interface {
	// Destination returns the destination for the given reflect value and column.
	Destination(rv reflect.Value, column []string) ([]any, error)
}

// rowDestination is a ColumnDestination which can be used to scan a row.
type rowDestination struct {
	// indexes stores the index of the column in the struct.
	// this could not be used to for deep struct scan.
	indexes [][]int

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

func (s *rowDestination) destinationForOneColumn(rv reflect.Value, columns []string) ([]any, error) {
	// if type is time.Time or implements sql.Scanner, we can scan it directly
	if rv.Type() == timeType || rv.Type().Implements(scannerType) {
		return []any{rv.Addr().Interface()}, nil
	}
	if rv.Kind() == reflect.Struct {
		return s.destinationForStruct(rv, columns)
	}
	// default behavior
	return []any{rv.Addr().Interface()}, nil
}

func (s *rowDestination) destination(rv reflect.Value, columns []string) ([]any, error) {
	if len(columns) == 1 {
		return s.destinationForOneColumn(rv, columns)
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
	for i, indexes := range s.indexes {
		if len(indexes) == 0 {
			dest[i] = new(any)
		} else {
			dest[i] = rv.FieldByIndex(indexes).Addr().Interface()
		}
	}
	return dest, nil
}

// setIndexes sets the indexes for the given reflect value and columns.
func (s *rowDestination) setIndexes(rv reflect.Value, columns []string) {
	tp := rv.Type()
	s.indexes = make([][]int, len(columns))
	s.findFromStruct(tp, columns, nil)
}

// findFromStruct finds the index from the given struct type.
func (s *rowDestination) findFromStruct(tp reflect.Type, columns []string, walk []int) {

	// finished is a helper function to check if the indexes completed or not.
	finished := func() bool {
		for i := range columns {
			if len(s.indexes[i]) == 0 {
				return false
			}
		}
		return true
	}

	// columnIndex is a map to store the index of the column.
	columnIndex := func() map[string]int {
		m := make(map[string]int)
		for i, column := range columns {
			m[column] = i
		}
		return m
	}()

	// walk into the struct
	for i := 0; i < tp.NumField(); i++ {
		// if we find all the columns destination, we can stop.
		if finished() {
			break
		}
		field := tp.Field(i)
		tag := field.Tag.Get("column")
		// if the tag is empty or "-", we can skip it.
		if skip := tag == "" && !field.Anonymous || tag == "-"; skip {
			continue
		}
		// if the field is anonymous and the type is struct, we can walk into it.
		if deepScan := field.Anonymous && field.Type.Kind() == reflect.Struct && len(tag) == 0; deepScan {
			s.findFromStruct(field.Type, columns, append(walk, i))
			continue
		}
		// find the index of the column
		index, ok := columnIndex[tag]
		if !ok {
			continue
		}
		// set the index
		s.indexes[index] = append(walk, field.Index...)
	}
}

var errRawBytesScan = errors.New("sql: RawBytes isn't allowed on scan")

func checkDestination(dest []any) error {
	for _, dp := range dest {
		if _, ok := dp.(*sql.RawBytes); ok {
			return errRawBytesScan
		}
	}
	return nil
}
