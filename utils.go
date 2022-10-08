package pillow

import (
	"errors"
	"reflect"
	"runtime"
	"strings"
)

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
