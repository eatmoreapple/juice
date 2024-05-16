package reflectlite

import (
	"reflect"
)

// TypeIdentify returns the string representation of the type, including the
// package path for non-built-in types.
func TypeIdentify[T any]() string {
	return typeToString(reflect.TypeOf((*T)(nil)).Elem())
}

// typeToString returns a string representation of the reflect.Type, including
// the package path for non-built-in types.
func typeToString(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Slice, reflect.Array, reflect.Ptr, reflect.Chan:
		// For these kinds, we want to print the element type.
		return t.Kind().String() + "[" + typeToString(t.Elem()) + "]"
	case reflect.Map:
		// For maps, we want to print both the key and value types.
		return "map[" + typeToString(t.Key()) + "]" + typeToString(t.Elem())
	case reflect.Struct, reflect.Interface:
		// For named struct and interface types, we include the package path.
		if t.Name() == "" {
			// This is an anonymous struct or interface, so we print the detailed struct/interface definition.
			return t.String()
		}
		return qualifiedName(t)
	default:
		// For other types (including basic types and named types), use the qualified name.
		return qualifiedName(t)
	}
}

// qualifiedName returns the name of the type with its package path if it's not a built-in type.
func qualifiedName(t reflect.Type) string {
	if t.PkgPath() != "" && t.Name() != "" {
		// The type has a package path and a name, so it's not a built-in type.
		return t.PkgPath() + "." + t.Name()
	}
	// It's a built-in type or unnamed type, just return the type's string representation.
	return t.String()
}
