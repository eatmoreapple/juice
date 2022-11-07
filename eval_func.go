package juice

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// return the length of the string or array
func length(v interface{}) int {
	switch v.(type) {
	case nil:
		return 0
	case string:
		return len(v.(string))
	default:
		rv := reflect.Indirect(reflect.ValueOf(v))
		switch rv.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
			return rv.Len()
		}
	}
	panic("len: invalid argument type")
}

// strSub returns a substring of the string.
// The first parameter is the string to be processed.
// The second parameter is the start position of the substring.
// The third parameter is the length of the substring.
func strSub(v interface{}, start, count int) string {
	if str, ok := v.(string); ok {
		return str[start : start+count]
	}
	panic("substr: invalid argument type")
}

// strJoin joins the elements of the array into a string.
// The first parameter is the array to be processed.
// The second parameter is the separator.
// Returns a string.
func strJoin(v interface{}, sep string) string {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Array, reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.String {
			builder := getBuilder()
			defer putBuilder(builder)
			for i := 0; i < rv.Len(); i++ {
				if i > 0 {
					builder.WriteString(sep)
				}
				builder.WriteString(rv.Index(i).String())
			}
			return builder.String()
		}
	}
	panic("join: invalid argument type")
}

// contains returns true if the value is in the array or string.
func contains(s interface{}, v interface{}) bool {
	switch s.(type) {
	case string:
		value, ok := v.(string)
		if !ok {
			v = fmt.Sprintf("%v", v)
		}
		return strings.Contains(s.(string), value)
	default:
		rv := reflect.Indirect(reflect.ValueOf(s))
		switch rv.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map:
			for i := 0; i < rv.Len(); i++ {
				if rv.Index(i).Interface() == v {
					return true
				}
			}
			return false
		}
	}
	panic("contains: invalid argument type")
}

// slice returns a slice of the array or string.
func slice(v interface{}, start, count int) []interface{} {
	rv := reflect.Indirect(reflect.ValueOf(v))
	switch rv.Kind() {
	case reflect.Array, reflect.Slice:
		result := rv.Slice(start, start+count)
		var ret []interface{}
		for i := 0; i < result.Len(); i++ {
			ret = append(ret, result.Index(i).Interface())
		}
		return ret
	}
	panic("slice: invalid argument type")
}

// title returns a copy of the string s with all Unicode letters that begin words mapped to their title case.
func title(text string) string {
	return strings.Title(text)
}

// lower returns a copy of the string s with all Unicode letters mapped to their lower case.
func lower(text string) string {
	return strings.ToLower(text)
}

// upper returns a copy of the string s with all Unicode letters mapped to their upper case.
func upper(text string) string {
	return strings.ToUpper(text)
}

// trim returns a slice of the string s with all leading and trailing Unicode code points contained in cutset removed.
func trim(text, cutest string) string {
	return strings.Trim(text, cutest)
}

// trimLeft returns a slice of the string s with all leading Unicode code points contained in cutset removed.
func trimLeft(text, cutest string) string {
	return strings.TrimLeft(text, cutest)
}

// trimRight returns a slice of the string s with all trailing Unicode code points contained in cutset removed.
func trimRight(text, cutest string) string {
	return strings.TrimRight(text, cutest)
}

// replace returns a copy of the string s with the first n non-overlapping instances of old replaced by new.
// If old is empty, it matches at the beginning of the string and after each UTF-8 sequence, yielding up to k+1 replacements for a k-rune string.
func replace(text, old, new string, n int) string {
	return strings.Replace(text, old, new, n)
}

// replaceAll returns a copy of the string s with all non-overlapping instances of old replaced by new.
// If old is empty, it matches at the beginning of the string and after each UTF-8 sequence, yielding up to k+1 replacements for a k-rune string.
func replaceAll(text, old, new string) string {
	return strings.ReplaceAll(text, old, new)
}

// split returns a slice of strings after splitting the string s at each instance of sep.
func split(text, sep string) []string {
	return strings.Split(text, sep)
}

// splitN returns a slice of strings after splitting the string s at each instance of sep, at most n times.
// If n == 0, SplitN returns an unlimited number of substrings.
// If n < 0, SplitN splits after each instance of sep.
func splitN(text, sep string, n int) []string {
	return strings.SplitN(text, sep, n)
}

// splitAfter returns a slice of strings after splitting the string s after each instance of sep.
func splitAfter(text, sep string) []string {
	return strings.SplitAfter(text, sep)
}

// RegisterEvalFunc registers a function for eval.
// The function must be a function with one return value.
// And Allowed to overwrite the built-in function.
func RegisterEvalFunc(name string, v interface{}) error {
	rv := reflect.Indirect(reflect.ValueOf(v))
	if rv.Kind() != reflect.Func {
		return errors.New("RegisterEvalFunc: v must be a function type")
	}
	if rv.Type().NumOut() != 1 {
		return errors.New("RegisterEvalFunc: v must be a function with one return value")
	}
	builtins[name] = rv
	return nil
}

// MustRegisterEvalFunc registers a function for eval.
// If an error occurs, it will panic.
func MustRegisterEvalFunc(name string, v interface{}) {
	err := RegisterEvalFunc(name, v)
	if err != nil {
		panic(err)
	}
}

// builtins is a map of built-in functions.
var builtins = map[string]reflect.Value{}

func init() {
	MustRegisterEvalFunc("len", length)
	MustRegisterEvalFunc("substr", strSub)
	MustRegisterEvalFunc("join", strJoin)
	MustRegisterEvalFunc("contains", contains)
	MustRegisterEvalFunc("slice", slice)
	MustRegisterEvalFunc("title", title)
	MustRegisterEvalFunc("lower", lower)
	MustRegisterEvalFunc("upper", upper)
	MustRegisterEvalFunc("trim", trim)
	MustRegisterEvalFunc("trimLeft", trimLeft)
	MustRegisterEvalFunc("trimRight", trimRight)
	MustRegisterEvalFunc("replace", replace)
	MustRegisterEvalFunc("replaceAll", replaceAll)
	MustRegisterEvalFunc("split", split)
	MustRegisterEvalFunc("splitN", splitN)
	MustRegisterEvalFunc("splitAfter", splitAfter)
}
