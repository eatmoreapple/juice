package juice

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
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

func runtimeFuncName(rv reflect.Value) string {
	// one id from function name
	name := runtime.FuncForPC(rv.Pointer()).Name()
	name = strings.ReplaceAll(strings.ReplaceAll(name, "/", "."), "*", "")
	return strings.TrimSuffix(name, "-fm")
}

// reflectValueToString converts reflect.Value to string
func reflectValueToString(v reflect.Value) string {
	if stringer, ok := v.Interface().(interface {
		String() string
	}); ok {
		return stringer.String()
	}
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	}
	return fmt.Sprintf("%v", v.Interface())
}
