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
	// Validate input is a pointer
	if rv.Kind() != reflect.Ptr {
		return ErrPointerRequired
	}

	// Check if there is any row and handle potential errors
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return fmt.Errorf("error occurred while fetching row: %w", err)
		}
		return sql.ErrNoRows
	}

	// Get column information
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	// Get the actual value to map into
	targetValue := reflect.Indirect(rv)

	// Create destination mapper
	columnDest := &rowDestination{}

	// Map columns to struct fields and create scan destinations
	dest, err := columnDest.Destination(targetValue, columns)
	if err != nil {
		return fmt.Errorf("failed to create destination mapping: %w", err)
	}

	// Scan row data into destinations
	if err = rows.Scan(dest...); err != nil {
		return fmt.Errorf("failed to scan row: %w", err)
	}

	// Check for any errors that occurred during row scanning
	if err = rows.Err(); err != nil {
		return fmt.Errorf("error occurred during row scanning: %w", err)
	}

	// Ensure there is only one row
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
// It maps the data from the SQL rows to the provided reflect.Value.
// The reflect.Value must be a pointer to a slice.
// Each row will be mapped to a new element in the slice.
func (m MultiRowsResultMap) MapTo(rv reflect.Value, rows *sql.Rows) error {
	if err := m.validateInput(rv); err != nil {
		return err
	}
	elementType := rv.Elem().Type().Elem()
	// get the element type and check if it's a pointer
	isPointer, isElementImplementsScanner := m.resolveTypes(elementType)

	// initialize element creator if not provided
	if m.New == nil {
		targetElementType := elementType
		if isPointer {
			targetElementType = targetElementType.Elem()
		}
		m.New = func() reflect.Value { return reflect.New(targetElementType) }
	}

	// map the rows to values
	values, err := m.mapRows(rows, isPointer, isElementImplementsScanner)
	if err != nil {
		return err
	}
	target := rv.Elem()
	// create slice with proper capacity and set the values
	result := reflect.MakeSlice(target.Type(), 0, len(values))
	target.Set(reflect.Append(result, values...))
	return nil
}

// validateInput validates that the input reflect.Value is a pointer to a slice
func (m MultiRowsResultMap) validateInput(rv reflect.Value) error {
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("%w: expected pointer to slice", ErrPointerRequired)
	}
	if rv.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("expected pointer to slice, got pointer to %v", rv.Elem().Kind())
	}
	return nil
}

// resolveTypes returns the element type, whether it's a pointer, and the actual type
func (m MultiRowsResultMap) resolveTypes(elementType reflect.Type) (bool, bool) {
	isPointer := elementType.Kind() == reflect.Ptr
	pointerType := elementType
	if !isPointer {
		pointerType = reflect.PointerTo(elementType)
	}
	return isPointer, pointerType.Implements(rowScannerType)
}

// mapRows maps the rows to a slice of reflect.Values
func (m MultiRowsResultMap) mapRows(rows *sql.Rows, isPointer bool, useScanner bool) ([]reflect.Value, error) {
	if useScanner {
		return m.mapWithRowScanner(rows, isPointer)
	}
	return m.mapWithColumnDestination(rows, isPointer)
}

// mapWithRowScanner maps rows using the RowScanner interface
func (m MultiRowsResultMap) mapWithRowScanner(rows *sql.Rows, isPointer bool) ([]reflect.Value, error) {
	// Pre-allocate slice with an initial capacity
	values := make([]reflect.Value, 0, 8)

	for rows.Next() {
		// Create a new instance. Since RowScanner is implemented with pointer receiver,
		// we always create a pointer type and use it directly for scanning
		newValue := m.New()
		if err := newValue.Interface().(RowScanner).ScanRows(rows); err != nil {
			return nil, fmt.Errorf("failed to scan row using RowScanner: %w", err)
		}

		if isPointer {
			values = append(values, newValue)
		} else {
			values = append(values, newValue.Elem())
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred while iterating rows: %w", err)
	}

	return values, nil
}

// mapWithColumnDestination maps rows using column destination
func (m MultiRowsResultMap) mapWithColumnDestination(rows *sql.Rows, isPointer bool) ([]reflect.Value, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}
	columnDest := &rowDestination{}
	// Pre-allocate slice with an initial capacity
	values := make([]reflect.Value, 0, 8)

	for rows.Next() {
		// Create a new instance and get its underlying value for column mapping
		newValue := m.New()
		elementValue := newValue.Elem()

		// Map database columns to struct fields and create scan destinations
		dest, err := columnDest.Destination(elementValue, columns)
		if err != nil {
			return nil, fmt.Errorf("failed to get destination: %w", err)
		}

		// Scan the current row into the destinations
		if err = rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Append either the pointer or the value based on the target type
		if isPointer {
			values = append(values, newValue)
		} else {
			values = append(values, elementValue)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred while iterating rows: %w", err)
	}

	return values, nil
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
	return nil, fmt.Errorf("expected struct, but got %s", rv.Kind())
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
