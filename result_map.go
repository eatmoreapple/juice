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
type MultiRowsResultMap struct {
	New func() reflect.Value
}

// MapTo implements ResultMapper interface.
// It maps the data from the SQL row to the provided reflect.Value.
// It maps each row to a new element in a slice.
func (m MultiRowsResultMap) MapTo(rv reflect.Value, rows *sql.Rows) error {

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
	if isPtr {
		el = el.Elem()
	}

	if m.New == nil {
		m.New = func() reflect.Value { return reflect.New(el) }
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
		nrv := m.New()

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

// ColumnDestination is a column destination which can be used to scan a row.
type ColumnDestination interface {
	// Destination returns the destination for the given reflect value and column.
	Destination(rv reflect.Value, column []string) ([]any, error)
}

// rowDestination implements ColumnDestination interface for mapping SQL query results
// to struct fields. It handles the mapping between database columns and struct fields
// by maintaining the field indexes and managing unmapped columns.
type rowDestination struct {
	// indexes stores the mapping between column positions and struct field indexes.
	// Each element is a slice of integers representing the path to the struct field:
	// - Empty slice means the column has no corresponding struct field
	// - Single integer means direct field access
	// - Multiple integers represent nested struct field access
	indexes [][]int

	// checked indicates whether the destination has been validated for sql.RawBytes.
	// This flag helps avoid redundant checks for the same rowDestination instance.
	checked bool

	// discard is a placeholder destination for SQL columns that don't have
	// corresponding struct fields. Each rowDestination instance maintains its
	// own discard variable to ensure thread safety during concurrent scans.
	discard any
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
			dest[i] = &s.discard
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

	// columnIndex is a map to store the index of the column.
	columnIndex := func() map[string]int {
		m := make(map[string]int)
		for i, column := range columns {
			m[column] = i
		}
		return m
	}()

	s.findFromStruct(tp, columns, columnIndex, nil)
}

// findFromStruct finds the index from the given struct type.
func (s *rowDestination) findFromStruct(tp reflect.Type, columns []string, columnIndex map[string]int, walk []int) {

	// finished is a helper function to check if the indexes completed or not.
	finished := func() bool {
		for i := range columns {
			if len(s.indexes[i]) == 0 {
				return false
			}
		}
		return true
	}

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
			s.findFromStruct(field.Type, columns, columnIndex, append(walk, i))
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
