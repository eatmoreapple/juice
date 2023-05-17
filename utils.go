package juice

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

// replacer defines the replacer of function name
var replacer = strings.NewReplacer("/", ".", "*", "", "(", "", ")", "")

// runtimeFuncName returns the function name of runtime
func runtimeFuncName(rv reflect.Value) string {
	// one id from function name
	name := runtime.FuncForPC(rv.Pointer()).Name()
	name = replacer.Replace(name)
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
