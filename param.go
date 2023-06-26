package juice

import (
	"context"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// Param is an alias of any type.
// It is used to represent the parameter of the statement and without type limitation.
type Param = any

type paramCtxKey struct{}

// CtxWithParam returns a new context with the parameter.
func CtxWithParam(ctx context.Context, param Param) context.Context {
	return context.WithValue(ctx, paramCtxKey{}, param)
}

// ParamFromContext returns the parameter from the context.
func ParamFromContext(ctx context.Context) Param {
	param, _ := ctx.Value(paramCtxKey{}).(Param)
	return param
}

// defaultParamKey is the default key of the parameter.
var defaultParamKey = func() string {
	// try to get the key from environment variable
	key := os.Getenv("JUICE_PARAM_KEY")
	// if not found, use the default key
	if len(key) == 0 {
		key = "param"
	}
	return key
}()

// Parameter is the interface that wraps the Get method.
// Get returns the value of the named parameter.
type Parameter interface {
	// Get returns the value of the named parameter with the type of reflect.Value.
	Get(name string) (reflect.Value, bool)
}

// make sure that ParamGroup implements Parameter.
var _ Parameter = (ParamGroup)(nil)

// ParamGroup is a group of parameters which implements the Parameter interface.
type ParamGroup []Parameter

// Get implements Parameter.
func (g ParamGroup) Get(name string) (reflect.Value, bool) {
	for _, p := range g {
		if p == nil {
			continue
		}
		if value, ok := p.Get(name); ok {
			return value, ok
		}
	}
	return reflect.Value{}, false
}

// make sure that structParameter implements Parameter.
var _ Parameter = (*structParameter)(nil)

// structParameter is a parameter that wraps a struct.
type structParameter struct {
	reflect.Value
}

// Get implements Parameter.
func (p structParameter) Get(name string) (reflect.Value, bool) {
	// try to one the value from field tag first
	tp := p.Type()
	for i := 0; i < p.NumField(); i++ {
		if tp.Field(i).Tag.Get(defaultParamKey) == name {
			return unwrapValue(p.Field(i)), true
		}
	}
	// if not found, try to one the value from field name
	value := p.FieldByNameFunc(func(search string) bool {
		// this might cause unexpected behavior
		return strings.EqualFold(name, search)
	})
	if !value.IsValid() {
		return reflect.Value{}, false
	}
	return value, true
}

// make sure that mapParameter implements Parameter.
var _ Parameter = (*mapParameter)(nil)

// mapParameter is a parameter that wraps a map.
type mapParameter struct {
	reflect.Value
}

// Get implements Parameter.
func (p mapParameter) Get(name string) (reflect.Value, bool) {
	value := p.MapIndex(reflect.ValueOf(name))
	if !value.IsValid() {
		return reflect.Value{}, false
	}
	return value, true
}

// make sure that sliceParameter implements Parameter.
var _ Parameter = (*sliceParameter)(nil)

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
	if !value.IsValid() {
		return reflect.Value{}, false
	}
	return value, true
}

// genericParameter is a parameter that wraps a generic value.
type genericParameter struct {
	reflect.Value

	// cache is used to cache the value of the parameter.
	cache map[string]reflect.Value
}

func (g *genericParameter) get(name string) (value reflect.Value, exists bool) {
	value = g.Value
	items := strings.Split(name, ".")
	var param Parameter
	for _, item := range items {

		// only unwrap when the value need to call Get method
		value = unwrapValue(value)

		// match the value type
		// only map, struct, slice and array can be wrapped as parameter
		switch value.Kind() {
		case reflect.Map:
			// if the map key is not a string type
			if value.Type().Key().Kind() != reflect.String {
				// TODO panic or return false?
				panic("the map key must be string type")
			}
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
	}
	return value, true
}

// Get implements Parameter.
// It will cache the value of the parameter for better performance.
func (g *genericParameter) Get(name string) (value reflect.Value, exists bool) {
	// try to get the value from cache first
	value, exists = g.cache[name]
	if exists {
		return value, exists
	}
	// if not found, then get the value from the generic parameter
	value, exists = g.get(name)
	if exists {
		if g.cache == nil {
			g.cache = make(map[string]reflect.Value)
		}
		// cache the value
		g.cache[name] = value
	}
	return value, exists
}

// newGenericParam creates a generic parameter.
// if the value is not a map, struct, slice or array, then wrap it as a map.
func newGenericParam(v any, wrapKey string) Parameter {
	if v == nil {
		return nil
	}
	value := reflect.ValueOf(v)

	tp := typeIndirect(value)

	switch tp {
	case reflect.Map, reflect.Struct, reflect.Slice, reflect.Array:
		// do nothing
	default:
		// if the value is not a map, struct, slice or array, then wrap it as a map
		if wrapKey == "" {
			wrapKey = defaultParamKey
		}
		value = reflect.ValueOf(H{wrapKey: v})
	}
	return &genericParameter{Value: value}
}

// NewParameter creates a new parameter with the given value.
func NewParameter(v Param) Parameter {
	return newGenericParam(v, "")
}

// H is a shortcut for map[string]any
type H map[string]any

// AsParam converts the H to a Parameter.
func (h H) AsParam() Parameter {
	return NewParameter(h)
}
