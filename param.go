package juice

import (
	"reflect"
	"strconv"
	"strings"
)

// Param is a map of string to reflect.Value
type Param map[string]reflect.Value

// Get returns the value of the key
// If the key is not found, it will return the default value
func (p Param) Get(name string) (reflect.Value, bool) {

	// split the name by dot
	// if the name is user.name, it will be split to user and name
	items := strings.Split(name, ".")

	var value reflect.Value

	// try to get the value from the split name
	for i, item := range items {

		// if it is the first item, try to get the value from the param
		// otherwise, try to get the value from the previous value

		if i == 0 {
			var exists bool
			value, exists = p[item]
			if !exists {
				return reflect.Value{}, false
			}
			continue
		}

		// if the previous value is not a struct, slice or a map, return false
		value = reflect.Indirect(value)

		switch value.Kind() {
		case reflect.Map:
			value = value.MapIndex(reflect.ValueOf(item))
		case reflect.Struct:
			field := value.FieldByName(item)
			if !field.IsValid() {
				// try to find it from tag
				for i := 0; i < value.NumField(); i++ {
					field := value.Type().Field(i)
					if field.Tag.Get("param") == item {
						value = value.Field(i)
						break
					}
				}
			} else {
				value = field
			}
		case reflect.Slice, reflect.Array:
			index, err := strconv.Atoi(item)
			if err != nil {
				return reflect.Value{}, false
			}
			value = value.Index(index)
		default:
			return reflect.Value{}, false
		}

		// if the value is not valid, return false
		if !value.IsValid() {
			return reflect.Value{}, false
		}

		// if the value is a pointer, get the value from the pointer
		for value.Kind() == reflect.Interface {
			value = value.Elem()
		}
	}

	return value, value.IsValid()
}

// ParamConverter is an interface that can convert itself to Param
type ParamConverter interface {
	ParamConvert() (Param, error)
}

const (
	defaultParamKey = "params"
	paramTag        = "param"
)

// ParamConvert converts any type to Param
func ParamConvert(v interface{}) (Param, error) {
	if v == nil {
		return make(Param), nil
	}
	if p, ok := v.(Param); ok {
		return p, nil
	}
	if p, ok := v.(ParamConverter); ok {
		return p.ParamConvert()
	}

	// get the value of the interface
	value := reflect.Indirect(reflect.ValueOf(v))
	switch value.Kind() {
	case reflect.Struct:
		return structConvert(value)
	case reflect.Map:
		return mapConvert(value)
	default:
		// if the value is not a struct or a map, try to get the value from the default key
		param := make(Param)
		param[defaultParamKey] = value
		return param, nil
	}
}

// mapConvert converts a map to Param
func mapConvert(value reflect.Value) (Param, error) {
	param := make(map[string]reflect.Value)
	for _, key := range value.MapKeys() {
		param[key.String()] = value.MapIndex(key)
	}
	return param, nil
}

// structConvert converts a struct to Param
func structConvert(value reflect.Value) (Param, error) {
	param := make(Param)
	for i := 0; i < value.NumField(); i++ {
		field := value.Type().Field(i)
		tag := field.Tag.Get(paramTag)
		if tag == "" {
			tag = field.Name
		}
		param[tag] = reflect.Indirect(value.Field(i))
	}
	return param, nil
}
