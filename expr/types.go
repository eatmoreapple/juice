package expr

import "reflect"

func isInt(r reflect.Value) bool {
	switch r.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	default:
		return false
	}
}

func isUint(r reflect.Value) bool {
	switch r.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func isFloat(r reflect.Value) bool {
	switch r.Kind() {
	case reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func isComplex(r reflect.Value) bool {
	switch r.Kind() {
	case reflect.Complex64, reflect.Complex128:
		return true
	default:
		return false
	}
}

func isString(r reflect.Value) bool {
	return r.Kind() == reflect.String
}

func isBool(r reflect.Value) bool {
	return r.Kind() == reflect.Bool
}
