package juice

import (
	"reflect"
	"strconv"
	"strings"
)

// defaultParamKey is the default key of the parameter.
const defaultParamKey = "param"

// Parameter is the interface that wraps the Get method.
// Get returns the value of the named parameter.
type Parameter interface {
	// Get returns the value of the named parameter with the type of reflect.Value.
	Get(name string) (reflect.Value, bool)
}

// ParamGroup is a group of parameters which implements the Parameter interface.
type ParamGroup []Parameter

// Get implements Parameter.
func (g ParamGroup) Get(name string) (reflect.Value, bool) {
	for _, p := range g {
		if value, ok := p.Get(name); ok {
			return value, ok
		}
	}
	return reflect.Value{}, false
}

// structParameter is a parameter that wraps a struct.
type structParameter struct {
	reflect.Value
}

// Get implements Parameter.
func (p structParameter) Get(name string) (reflect.Value, bool) {
	// try to one the value from field tag first
	for i := 0; i < p.NumField(); i++ {
		field := p.Type().Field(i)
		if field.Tag.Get("param") == name {
			return p.Field(i), true
		}
	}
	// if not found, try to one the value from field name
	value := p.FieldByNameFunc(func(search string) bool {
		// this might cause unexpected behavior
		return strings.EqualFold(name, search)
	})
	return value, value.IsValid()
}

// mapParameter is a parameter that wraps a map.
type mapParameter struct {
	reflect.Value
}

// Get implements Parameter.
func (p mapParameter) Get(name string) (reflect.Value, bool) {
	value := p.MapIndex(reflect.ValueOf(name))
	return value, value.IsValid()
}

// sliceParameter is a parameter that wraps a slice.
type sliceParameter struct {
	reflect.Value
}

// Get implements Parameter.
func (p sliceParameter) Get(name string) (reflect.Value, bool) {
	index, err := strconv.Atoi(name)
	if err != nil {
		return reflect.Value{}, false
	}
	value := p.Index(index)
	return value, value.IsValid()
}

// genericParameter is a parameter that wraps a generic value.
type genericParameter struct {
	reflect.Value
}

func (g *genericParameter) Get(name string) (value reflect.Value, exists bool) {
	value = g.Value
	items := strings.Split(name, ".")
	var param Parameter
	for _, item := range items {
		// match the value type
		// if the value is a map, then use mapParameter
		// if the value is a struct, then use structParameter
		// if the value is a slice or array, then use sliceParameter
		// otherwise, return false
		switch value.Kind() {
		case reflect.Map:
			param = mapParameter{value}
		case reflect.Struct:
			param = structParameter{value}
		case reflect.Slice, reflect.Array:
			param = sliceParameter{value}
		default:
			// otherwise, return false
			return reflect.Value{}, false
		}
		value, exists = param.Get(item)
		if !exists {
			return reflect.Value{}, false
		}

		// if the value is a pointer, then dereference it
		value = reflect.Indirect(value)

		// if the value is an interface, then unwrap it
		for value.Kind() == reflect.Interface {
			value = value.Elem()
		}
	}
	return value, true
}

// newGenericParam creates a generic parameter.
// if the value is not a map, struct, slice or array, then wrap it as a map.
func newGenericParam(v any, wrapKey string) Parameter {
	if v == nil {
		return nil
	}
	value := reflect.Indirect(reflect.ValueOf(v))
	switch value.Kind() {
	case reflect.Map, reflect.Struct, reflect.Slice, reflect.Array:
	default:
		// if the value is not a map, struct, slice or array, then wrap it as a map
		if wrapKey == "" {
			wrapKey = defaultParamKey
		}
		value = reflect.ValueOf(H{wrapKey: v})
	}
	return &genericParameter{value}
}

// H is a shortcut for map[string]any
type H map[string]any

func (h H) AsParam() Parameter {
	return newGenericParam(h, "")
}
