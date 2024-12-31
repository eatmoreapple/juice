package juice

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/go-juicedev/juice/internal/reflectlite"
)

// replacer defines the replacer of function name
var replacer = strings.NewReplacer("/", ".", `\`, ".", "*", "", "(", "", ")", "")

// runtimeFuncName returns the function name of runtime
func runtimeFuncName(rv reflect.Value) string {
	// one id from function name
	name := runtime.FuncForPC(rv.Pointer()).Name()
	name = replacer.Replace(name)
	return strings.TrimSuffix(name, "-fm")
}

// reflectValueToString converts reflect.Value to string
func reflectValueToString(v reflect.Value) string {
	v = reflectlite.Unwrap(v)
	switch t := v.Interface().(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(v.Bytes())
	case fmt.Stringer:
		return t.String()
	case int, int8, int16, int32, int64:
		return strconv.FormatInt(v.Int(), 10)
	case uint, uint8, uint16, uint32, uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case float32:
		return strconv.FormatFloat(v.Float(), 'g', -1, 32)
	case float64:
		return strconv.FormatFloat(v.Float(), 'g', -1, 64)
	case bool:
		return strconv.FormatBool(v.Bool())
	default:
		return fmt.Sprintf("%v", t)
	}
}
