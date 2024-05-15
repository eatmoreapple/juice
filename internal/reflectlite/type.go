package reflectlite

import "reflect"

// TypeIdentify returns the string representation of the type.
func TypeIdentify[T any]() string {
	rt := reflect.TypeOf((*T)(nil)).Elem()
	name := rt.String()
	star := ""
	if rt.Name() == "" {
		if pt := rt; pt.Kind() == reflect.Pointer {
			star = "*"
			rt = pt
		}
	}
	if rt.Name() != "" {
		if rt.PkgPath() == "" {
			name = star + rt.Name()
		} else {
			name = star + rt.PkgPath() + "." + rt.Name()
		}
	}
	return name
}
