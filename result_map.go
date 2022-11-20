package juice

import (
	"fmt"
	"reflect"
)

type ResultMap interface {
	ColumnValue(rv reflect.Value, column string) (reflect.Value, bool, error)
}

// resultMap implements ResultMapper interface
type resultMap struct {
	id           string
	results      resultGroup
	associations associationGroup
	mapping      map[string][]string
}

// init initializes resultMap
func (r *resultMap) init() error {
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

// ID returns id of resultMap.
func (r *resultMap) ID() string {
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

// ColumnValue implements ResultMapper interface.
func (r *resultMap) ColumnValue(rv reflect.Value, column string) (reflect.Value, bool, error) {
	properties, ok := r.mapping[column]
	if !ok {
		return reflect.Value{}, false, nil
	}
	for _, v := range properties {
		rv = rv.FieldByName(v)
		if !rv.IsValid() {
			return reflect.Value{}, false, fmt.Errorf("field %s is invalid", v)
		}
	}
	return rv, true, nil
}

// IndexResultMap is a ResultMap that uses indexes to find the field.
type IndexResultMap map[string][]int

// ColumnValue implements ResultMapper interface.
func (m IndexResultMap) ColumnValue(rv reflect.Value, column string) (reflect.Value, bool, error) {
	indexes, ok := m[column]
	if !ok {
		return reflect.Value{}, false, nil
	}
	for _, v := range indexes {
		rv = rv.Field(v)
		if !rv.IsValid() {
			return reflect.Value{}, false, fmt.Errorf("column `%s` index %d is invalid", column, v)
		}
	}
	return rv, true, nil
}

// newKeyValueResultMap creates a new KeyValueResultMap with the given reflect value.
func newKeyValueResultMap(rv reflect.Value) (KeyValueResultMap, error) {
	var dest = make(map[string]reflect.Value)
	if rv.Kind() == reflect.Struct {
		for i := 0; i < rv.NumField(); i++ {
			field := rv.Field(i)

			// skip unexported
			if !field.CanSet() {
				continue
			}

			tag := rv.Type().Field(i).Tag.Get("column")

			// is deep struct
			if field.Kind() == reflect.Struct && tag == "" && !field.Type().Implements(scannerType) {
				// recursive call
				mapping, err := newKeyValueResultMap(field)
				if err != nil {
					return nil, err
				}
				for k, v := range mapping {
					if _, ok := dest[k]; ok {
						return nil, fmt.Errorf("field name %s is unbiguous", k)
					}
					dest[k] = v
				}
			} else {
				// skip field with no tag
				if tag == "" {
					continue
				}
				if _, ok := dest[tag]; ok {
					return nil, fmt.Errorf("field name %s is unbiguous", tag)
				}
				dest[tag] = field
			}
		}
	}
	return dest, nil
}

// KeyValueResultMap is a ResultMap that uses key value to find the field.
type KeyValueResultMap map[string]reflect.Value

// ColumnValue implements ResultMapper interface.
func (m KeyValueResultMap) ColumnValue(rv reflect.Value, column string) (reflect.Value, bool, error) {
	v, ok := m[column]
	if !ok {
		return reflect.Value{}, false, nil
	}
	return v, true, nil
}

// newIndexResultMap is a helper function to scan rows to given entity
// it will return a map of column name to index of the field
func newIndexResultMap(rv reflect.Value) (IndexResultMap, error) {
	var dest = make(map[string][]int)
	if rv.Kind() == reflect.Struct {
		for i := 0; i < rv.NumField(); i++ {
			field := rv.Field(i)

			// skip unexported field
			if !field.CanSet() {
				continue
			}

			tag := rv.Type().Field(i).Tag.Get("column")

			// is deep struct
			if field.Kind() == reflect.Struct && tag == "" && !field.Type().Implements(scannerType) {
				mapping, err := newIndexResultMap(field)
				if err != nil {
					return nil, err
				}
				for k, v := range mapping {
					if _, ok := dest[k]; ok {
						return nil, fmt.Errorf("field name %s is unbiguous", k)
					}
					dest[k] = append([]int{i}, v...)
				}
			} else {
				if tag == "" {
					continue
				}
				if _, ok := dest[tag]; ok {
					return nil, fmt.Errorf("field name %s is unbiguous", tag)
				}
				dest[tag] = []int{i}
			}
		}
	}
	return dest, nil
}
