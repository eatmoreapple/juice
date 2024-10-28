/*
Copyright 2024 eatmoreapple

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
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/eatmoreapple/juice/internal/reflectlite"
)

// selectKeyGenerator is an interface that defines a method to generate keys for a given reflect.Value.
type selectKeyGenerator interface {
	GenerateKeyTo(v reflect.Value) error
}

// findFieldIndexesFromProperties finds the field indexes in a struct type based on the provided key properties.
// It returns a slice of indexes and a boolean indicating if the indexes were found.
func findFieldIndexesFromProperties(t reflect.Type, keyProperties ...string) ([]int, bool) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, false
	}
	if len(keyProperties) == 0 {
		return reflectlite.TypeFrom(t).GetFieldIndexesFromTag("autoincr", "true")
	}
	var indexes []int
	for _, keyProperty := range keyProperties {
		// try to find the field from the given struct.
		// if isPublic is true, then it means the following keyProperties are the field names.
		// otherwise, the following keyProperties are the tag names.
		isPublic := unicode.IsUpper(rune(keyProperty[0]))
		if isPublic {
			structField, ok := t.FieldByName(keyProperty)
			if !ok {
				return nil, false
			}
			t = structField.Type
			indexes = append(indexes, structField.Index...)
		} else {
			fieldIndexes, ok := reflectlite.TypeFrom(t).GetFieldIndexesFromTag("column", keyProperty)
			if !ok {
				return nil, false
			}
			t = t.FieldByIndex(fieldIndexes).Type
			indexes = append(indexes, fieldIndexes...)
		}
	}
	return indexes, len(indexes) > 0
}

// singleKeyGenerator is a struct that holds an id and a key property for generating keys.
type singleKeyGenerator struct {
	id          int64
	keyProperty string
}

// GenerateKeyTo generates a key for the given reflect.Value based on the key property and sets it to the id.
// It returns an error if the operation fails.
func (s singleKeyGenerator) GenerateKeyTo(v reflect.Value) error {
	if v.Kind() != reflect.Ptr {
		return ErrPointerRequired
	}
	v = reflect.Indirect(v)
	if v.Kind() != reflect.Struct {
		return errors.New("the param is not a struct")
	}
	// find the field indexes based on the key property
	var indexes []int
	if len(s.keyProperty) > 0 {
		indexes, _ = findFieldIndexesFromProperties(v.Type(), strings.Split(s.keyProperty, ".")...)
	} else {
		indexes, _ = findFieldIndexesFromProperties(v.Type())
	}
	if len(indexes) == 0 {
		return nil
	}
	v = v.FieldByIndex(indexes)
	if !v.IsValid() {
		return errors.New("invalid id")
	}
	if !v.CanInt() {
		return fmt.Errorf("the keyProperty %s is not a int", s.keyProperty)
	}
	v.SetInt(s.id)
	return nil
}

// batchKeyGenerator is a struct that holds an id, a key property, and a key increment for generating keys in batch.
type batchKeyGenerator struct {
	id           int64
	keyProperty  string
	keyIncrement int64
}

// GenerateKeyTo generates keys for each element in the given reflect.Value slice based on the key property and sets them to the id.
// It returns an error if the operation fails.
func (s batchKeyGenerator) GenerateKeyTo(v reflect.Value) error {
	v = reflect.Indirect(v)

	// ensure the param is a slice or array
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
	default:
		return errors.New("the param is not a slice or array")
	}

	// slice or array element type
	elementType := v.Type().Elem()
	isPrt := elementType.Kind() == reflect.Ptr

	if isPrt {
		elementType = elementType.Elem()
	}
	if elementType.Kind() != reflect.Struct {
		return errors.New("the element of the slice or array is not a struct")
	}
	// find the field indexes based on the key property
	var indexes []int
	if len(s.keyProperty) > 0 {
		indexes, _ = findFieldIndexesFromProperties(elementType, strings.Split(s.keyProperty, ".")...)
	} else {
		indexes, _ = findFieldIndexesFromProperties(elementType)
	}
	if len(indexes) == 0 {
		return nil
	}
	length := v.Len()
	pk := s.id
	for i := length - 1; i >= 0; i-- {
		value := v.Index(i)
		if isPrt {
			value = value.Elem()
		}
		// try to find the field indexes based on the key property
		// and ensure the field is valid and can be converted to int
		value = value.FieldByIndex(indexes)
		if !value.IsValid() {
			return fmt.Errorf("invalid field %s", s.keyProperty)
		}
		if !value.CanInt() {
			return fmt.Errorf("can not convert %s to int", s.keyProperty)
		}
		if i != length-1 {
			pk -= s.keyIncrement
		}
		value.SetInt(pk)
	}
	return nil
}
