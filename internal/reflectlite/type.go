package reflectlite

import (
	"reflect"
)

// IndirectType returns the type of the element if the type is a pointer type.
func IndirectType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
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

// TypeIdentify returns the string representation of the type, including the
// package path for non-built-in types.
func TypeIdentify[T any]() string {
	return typeToString(reflect.TypeOf((*T)(nil)).Elem())
}

type Type struct {
	reflect.Type
}

// Identify returns the string representation of the type, including the
// package path for non-built-in types.
func (t Type) Identify() string {
	return typeToString(t.Type)
}

// Indirect returns the type of the element if the type is a pointer type.
func (t Type) Indirect() Type {
	return Type{IndirectType(t.Type)}
}

func getFieldIndexesFromTag(value reflect.Type, tagName, tagValue string) ([]int, bool) {
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.Type.Kind() == reflect.Struct && field.Tag.Get(tagName) == "" {
			if indexes, ok := getFieldIndexesFromTag(field.Type, tagName, tagValue); ok {
				return append(field.Index[:], indexes...), ok
			} else {
				continue
			}
		}
		if tag := field.Tag.Get(tagName); tag == tagValue {
			return field.Index[:], true
		}
	}
	return nil, false
}

func (t Type) GetFieldIndexesFromTag(tagName, tagValue string) ([]int, bool) {
	if t.Kind() != reflect.Struct {
		return nil, false
	}
	return getFieldIndexesFromTag(t.Type, tagName, tagValue)
}

// TypeFrom returns a Type from the given reflect.Type.
func TypeFrom(t reflect.Type) Type {
	return Type{t}
}
