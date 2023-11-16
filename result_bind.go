package juice

import (
	"errors"
	"fmt"
	"reflect"
)

// BinderRouter is a map that associates a string key to any value.
type BinderRouter map[string]any

// ResultBinder is an interface that defines a single method BindTo.
// This method takes a reflect.Value and returns a BinderRouter and an error.
type ResultBinder interface {
	BindTo(v reflect.Value) (BinderRouter, error)
}

// ResultBinderGroup is a slice of ResultBinders.
type ResultBinderGroup []ResultBinder

// BindTo method for a group of ResultBinders. It iterates over each binder in the group,
// calls its BindTo method, and merges the results into a single BinderRouter.
// If a key is found in more than one binder, it returns an error.
func (r ResultBinderGroup) BindTo(v reflect.Value) (BinderRouter, error) {
	var result = make(BinderRouter)
	for _, binder := range r {
		router, err := binder.BindTo(v)
		if err != nil {
			return nil, err
		}
		for key := range router {
			if _, ok := result[key]; ok {
				return nil, fmt.Errorf("duplicate key %s", key)
			}
			result[key] = router[key]
		}
	}
	return result, nil
}

// propertyResultBinder is a struct that contains a column and a property string.
type propertyResultBinder struct {
	column   string
	property string
}

// BindTo method for a propertyResultBinder. It checks if the provided reflect.Value is a pointer to a struct,
// then it finds the field with the name of the property in the struct and returns a BinderRouter that associates the column to the address of the field.
func (p *propertyResultBinder) BindTo(v reflect.Value) (BinderRouter, error) {
	if v.Kind() != reflect.Ptr {
		return nil, errors.New("result must be a pointer")
	}
	v = reflect.Indirect(v)
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("result must be a struct, but got %s", v.Kind())
	}
	field := v.FieldByName(p.property)
	if !field.IsValid() {
		return nil, fmt.Errorf("property %s not found", p.property)
	}
	return BinderRouter{p.column: field.Addr().Interface()}, nil
}

// fromResultNode is a function that takes a resultNode and returns a ResultBinder.
func fromResultNode(r resultNode) ResultBinder {
	return &propertyResultBinder{column: r.column, property: r.property}
}

// fromResultNodeGroup is a function that takes a resultGroup and returns a ResultBinderGroup.
func fromResultNodeGroup(rs resultGroup) ResultBinderGroup {
	group := make(ResultBinderGroup, 0, len(rs))
	for _, r := range rs {
		group = append(group, fromResultNode(*r))
	}
	return group
}

// associationResultBinder is a struct that contains a slice of ResultBinders and a property string.
type associationResultBinder struct {
	binders  []ResultBinder
	property string
}

// BindTo method for an associationResultBinder. It checks if the provided reflect.Value is a pointer to a struct,
// then it finds the field with the name of the property in the struct and calls the BindTo method for each binder in the associationResultBinder.
// It merges the results into a single BinderRouter. If a key is found in more than one binder, it returns an error.
func (a *associationResultBinder) BindTo(v reflect.Value) (BinderRouter, error) {
	if v.Kind() != reflect.Ptr {
		return nil, errors.New("result must be a pointer")
	}
	v = reflect.Indirect(v)
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("result must be a struct, but got %s", v.Kind())
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
	var result = make(BinderRouter)

	for _, binder := range a.binders {
		router, err := binder.BindTo(field.Addr())
		if err != nil {
			return nil, err
		}
		for key := range router {
			if _, ok := result[key]; ok {
				return nil, fmt.Errorf("duplicate key %s", key)
			}
			result[key] = router[key]
		}
	}
	return result, nil
}

// fromAssociation is a function that takes an association and returns a ResultBinder.
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

// fromAssociationGroup is a function that takes an associationGroup and returns a ResultBinderGroup.
func fromAssociationGroup(list associationGroup) ResultBinderGroup {
	group := make(ResultBinderGroup, 0, len(list))
	for _, a := range list {
		group = append(group, fromAssociation(*a))
	}
	return group
}

// collectionResultBinder is a struct that contains a slice of ResultBinders and a property string.
type collectionResultBinder struct {
	binders  []ResultBinder
	property string
}

// BindTo method for a collectionResultBinder. It checks if the provided reflect.Value is a pointer to a struct,
// then it finds the field with the name of the property in the struct and calls the BindTo method for each binder in the collectionResultBinder.
// It merges the results into a single BinderRouter. If a key is found in more than one binder, it returns an error.
func (c *collectionResultBinder) BindTo(v reflect.Value) (BinderRouter, error) {
	if v.Kind() != reflect.Ptr {
		return nil, errors.New("result must be a pointer")
	}
	v = reflect.Indirect(v)
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("result must be a struct, but got %s", v.Kind())
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

	var result = make(BinderRouter)

	for _, binder := range c.binders {
		router, err := binder.BindTo(instance)
		if err != nil {
			return nil, err
		}
		for key := range router {
			if _, ok := result[key]; ok {
				return nil, fmt.Errorf("duplicate key %s", key)
			}
			result[key] = router[key]
		}
	}
	field.Set(reflect.Append(field, instance))

	return result, nil
}

// fromCollection is a function that takes a collection and returns a ResultBinder.
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

// fromCollectionGroup is a function that takes a collectionGroup and returns a ResultBinderGroup.
func fromCollectionGroup(list collectionGroup) ResultBinderGroup {
	group := make(ResultBinderGroup, 0, len(list))
	for _, c := range list {
		group = append(group, fromCollection(*c))
	}
	return group
}
