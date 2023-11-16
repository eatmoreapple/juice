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

// ErrTooManyRows is returned when the result set has too many rows but excepted only one row.
var ErrTooManyRows = errors.New("juice: too many rows in result set")

type ResultMap interface {
	ResultTo(rv reflect.Value, row *sql.Rows) error
}

// RowResultMap is a ResultMap that maps a rowDestination to a non-slice type.
type RowResultMap struct{}

// ResultTo implements ResultMapper interface.
func (RowResultMap) ResultTo(rv reflect.Value, rows *sql.Rows) error {
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

// ResultTo implements ResultMapper interface.
func (r *resultMapNode) ResultTo(rv reflect.Value, rows *sql.Rows) error {
	if rv.Kind() != reflect.Ptr {
		return ErrPointerRequired
	}
	rv = reflect.Indirect(rv)

	if rv.Kind() == reflect.Slice {
		return r.resultToSlice(rv, rows)
	}
	return r.resultToStruct(rv, rows)
}

func (r *resultMapNode) binderToStruct(rv reflect.Value, rows *sql.Rows) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	items, err := r.binders.BindTo(rv.Addr())
	if err != nil {
		return err
	}
	dest := make([]any, len(columns))

	columnMap := func() map[string]int {
		var m = make(map[string]int)
		for i, column := range columns {
			m[column] = i
		}
		return m
	}()

	for _, item := range items {
		if index, ok := columnMap[item.Column]; ok {
			dest[index] = item.Addr
		}
	}
	return rows.Scan(dest...)
}

// resultToStruct scans rows to struct with given entity
// it's used when the relation is one to one or one to many
func (r *resultMapNode) resultToStruct(rv reflect.Value, rows *sql.Rows) error {
	if rv.Kind() != reflect.Struct {
		return errors.New("slice element must be a struct")
	}
	for rows.Next() {
		if err := r.binderToStruct(rv, rows); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}

func (r *resultMapNode) getValuesFromRows(rows *sql.Rows, el reflect.Type, isPtr bool) ([]reflect.Value, error) {
	values := make([]reflect.Value, 0)
	for rows.Next() {
		instance := reflect.New(el)
		value := instance.Elem()
		if err := r.binderToStruct(value, rows); err != nil {
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
		pk := directValue.FieldByName(r.pk.property)
		var found bool
		for i := 0; i < ret.Len(); i++ {
			current := reflect.Indirect(ret.Index(i))
			field := current.FieldByName(r.pk.property)
			if found = reflect.Indirect(field).Interface() == reflect.Indirect(pk).Interface(); found {
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

func checkDestination(dest []any) error {
	for _, dp := range dest {
		if _, ok := dp.(*sql.RawBytes); ok {
			return errors.New("sql: RawBytes isn't allowed on scan")
		}
	}
	return nil
}

type BinderItem struct {
	Addr   any
	Column string
}

type ResultBinder interface {
	BindTo(v reflect.Value) ([]BinderItem, error)
}

type ResultBinderGroup []ResultBinder

func (r ResultBinderGroup) BindTo(v reflect.Value) ([]BinderItem, error) {
	var binders = make([]BinderItem, 0)
	for _, binder := range r {
		items, err := binder.BindTo(v)
		if err != nil {
			return nil, err
		}
		binders = append(binders, items...)
	}
	return binders, nil
}

type propertyResultBinder struct {
	column   string
	property string
}

func (p *propertyResultBinder) BindTo(v reflect.Value) ([]BinderItem, error) {
	if v.Kind() != reflect.Ptr {
		return nil, errors.New("result must be a pointer")
	}
	v = reflect.Indirect(v)
	if v.Kind() != reflect.Struct {
		return nil, errors.New("result must be a struct")
	}
	field := v.FieldByName(p.property)
	if !field.IsValid() {
		return nil, fmt.Errorf("property %s not found", p.property)
	}
	item := BinderItem{Addr: field.Addr().Interface(), Column: p.column}
	return []BinderItem{item}, nil
}

func fromResultNode(r resultNode) ResultBinder {
	return &propertyResultBinder{column: r.column, property: r.property}
}

func fromResultNodeGroup(rs resultGroup) ResultBinderGroup {
	group := make(ResultBinderGroup, 0, len(rs))
	for _, r := range rs {
		group = append(group, fromResultNode(*r))
	}
	return group
}

type associationResultBinder struct {
	binders  []ResultBinder
	property string
}

func (a *associationResultBinder) BindTo(v reflect.Value) ([]BinderItem, error) {
	if v.Kind() != reflect.Ptr {
		return nil, errors.New("result must be a pointer")
	}
	v = reflect.Indirect(v)
	if v.Kind() != reflect.Struct {
		return nil, errors.New("result must be a struct")
	}
	field := v.FieldByName(a.property)
	if !field.IsValid() {
		return nil, fmt.Errorf("property %s not found", a.property)
	}
	if field.Kind() == reflect.Ptr && field.Elem().Kind() == reflect.Struct {
		field.Set(reflect.New(field.Type()))
		field = field.Elem()
	} else if field.Kind() != reflect.Struct {
		return nil, fmt.Errorf("property %s must be a struct", a.property)
	}
	var binders = make([]BinderItem, 0, len(a.binders))
	for _, binder := range a.binders {
		item, err := binder.BindTo(field.Addr())
		if err != nil {
			return nil, err
		}
		binders = append(binders, item...)
	}
	return binders, nil
}

func fromAssociation(association association) ResultBinder {
	var binders = make([]ResultBinder, 0, len(association.results))
	for _, result := range association.results {
		binders = append(binders, fromResultNode(*result))
	}
	for _, association := range association.associations {
		binders = append(binders, fromAssociation(*association))
	}
	return &associationResultBinder{binders: binders, property: association.property}
}

func fromAssociationGroup(list associationGroup) ResultBinderGroup {
	group := make(ResultBinderGroup, 0, len(list))
	for _, a := range list {
		group = append(group, fromAssociation(*a))
	}
	return group
}

type collectionResultBinder struct {
	binders  []ResultBinder
	property string
}

func (c *collectionResultBinder) BindTo(v reflect.Value) ([]BinderItem, error) {
	if v.Kind() != reflect.Ptr {
		return nil, errors.New("result must be a pointer")
	}
	v = reflect.Indirect(v)
	if v.Kind() != reflect.Struct {
		return nil, errors.New("result must be a struct")
	}
	field := v.FieldByName(c.property)
	if !field.IsValid() {
		return nil, fmt.Errorf("property %s not found", c.property)
	}
	if field.Kind() != reflect.Slice {
		return nil, fmt.Errorf("property %s must be a slice", c.property)
	}
	elem := field.Type().Elem()
	if elem.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("property %s must be a slice of pointer", c.property)
	}

	instance := reflect.New(elem.Elem())

	var binders = make([]BinderItem, 0, len(c.binders))
	for _, binder := range c.binders {
		item, err := binder.BindTo(instance)
		if err != nil {
			return nil, err
		}
		binders = append(binders, item...)
	}
	field.Set(reflect.Append(field, instance))
	return binders, nil
}

func fromCollection(collection collection) ResultBinder {
	var binders = make([]ResultBinder, 0, len(collection.resultGroup))
	for _, result := range collection.resultGroup {
		binders = append(binders, fromResultNode(*result))
	}
	for _, association := range collection.associationGroup {
		binders = append(binders, fromAssociation(*association))
	}
	return &collectionResultBinder{binders: binders, property: collection.property}
}

func fromCollectionGroup(list collectionGroup) ResultBinderGroup {
	group := make(ResultBinderGroup, 0, len(list))
	for _, c := range list {
		group = append(group, fromCollection(*c))
	}
	return group
}
