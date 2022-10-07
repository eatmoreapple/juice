package pillow

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

func ParamConvert(v interface{}) (map[string]reflect.Value, error) {
	if v == nil {
		return make(map[string]reflect.Value), nil
	}
	value := reflect.Indirect(reflect.ValueOf(v))
	switch value.Kind() {
	case reflect.Struct:
		return structConvert(value)
	case reflect.Map:
		return mapConvert(value)
	default:
		return nil, fmt.Errorf("unsupported type %s", value.Kind())
	}
}

func mapConvert(value reflect.Value) (map[string]reflect.Value, error) {
	param := make(map[string]reflect.Value)
	for _, key := range value.MapKeys() {
		param[key.String()] = value.MapIndex(key)
	}
	return param, nil
}

func structConvert(value reflect.Value) (map[string]reflect.Value, error) {
	param := make(map[string]reflect.Value)
	for i := 0; i < value.NumField(); i++ {
		field := value.Type().Field(i)
		tag := field.Tag.Get("param")
		if tag == "" {
			tag = field.Name
		}
		param[tag] = value.Field(i)
	}
	return param, nil
}

func underlineToCamel(text string) string {
	var result = make([]byte, 0, len(text))
	for i := 0; i < len(text); i++ {
		if i == 0 {
			result = append(result, text[i]-32)
			continue
		}
		if text[i] == '_' {
			i++
			if i < len(text) {
				result = append(result, text[i]-32)
			}
		} else {
			result = append(result, text[i])
		}
	}
	return string(result[:])
}

func FuncForPC(v any) (string, error) {
	value := reflect.ValueOf(v)

	// Check if the value is a function
	if value.Kind() != reflect.Func {
		return "", errors.New("v must be a function")
	}

	// get id from function name
	name := runtime.FuncForPC(value.Pointer()).Name()
	name = strings.ReplaceAll(strings.ReplaceAll(name, "/", "."), "*", "")

	return strings.TrimSuffix(name, "-fm"), nil
}
