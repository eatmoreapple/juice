package internal

import (
	"fmt"
	"go/format"
	"sort"
	"strings"
)

type Value struct {
	// Name is a name of value.
	Name string
	// Type is a type of value.
	Type string
	// Import is a import of value.
	Import Import
	// IsPointer is a flag of pointer.
	IsPointer bool
	// IsStruct is a flag of struct.
	IsStruct bool
	// IsMap is a flag of map.
	IsMap bool
	// IsSlice is a flag of slice.
	IsSlice bool
}

func (v Value) TypeName() string {
	var name string
	if v.Import.Name != "" {
		if v.IsSlice {
			if v.IsPointer {
				name = fmt.Sprintf("[]*%s.%s", v.Import.Name, v.Type)
			} else {
				name = fmt.Sprintf("[]%s.%s", v.Import.Name, v.Type)
			}
			return name
		} else if v.IsMap {
			if v.IsPointer {
				name = fmt.Sprintf("map[string]*%s.%s", v.Import.Name, v.Type)
			} else {
				name = fmt.Sprintf("map[string]%s.%s", v.Import.Name, v.Type)
			}
			return name
		} else {
			name = v.Import.Name + "." + v.Type
		}
	} else {
		if v.IsSlice {
			if v.IsPointer {
				name = fmt.Sprintf("[]*%s", v.Type)
			} else {
				name = fmt.Sprintf("[]%s", v.Type)
			}
			return name
		} else if v.IsMap {
			if v.IsPointer {
				name = fmt.Sprintf("map[string]*%s", v.Type)
			} else {
				name = fmt.Sprintf("map[string]%s", v.Type)
			}
			return name
		} else {
			name = v.Type
		}
	}
	if v.IsPointer {
		return "*" + name
	}
	return name
}

func (v Value) String() string {
	if v.Name == "" {
		return v.TypeName()
	}
	return fmt.Sprintf("%s %s", v.Name, v.TypeName())
}

type Values []Value

func (v Values) Imports() Imports {
	imports := make(map[string]Import)
	for _, value := range v {
		if value.Import.Path != "" {
			imports[value.Import.Path] = value.Import
		}
	}
	var result = make(Imports, 0, len(imports))
	for _, value := range imports {
		result = append(result, value)
	}
	sort.Sort(result)
	return result
}

func (v Values) String() string {
	vs := make([]string, 0, len(v))
	for _, value := range v {
		vs = append(vs, value.String())
	}
	return fmt.Sprintf("(%s)", strings.Join(vs, ", "))
}

type Import struct {
	Path string
	Name string
}

func (i Import) HasAlias() bool {
	path := strings.Trim(i.Path, `"`)
	pkg := strings.Split(path, "/")
	return pkg[len(pkg)-1] != i.Name
}

type Imports []Import

func (o Imports) Len() int {
	return len(o)
}

func (o Imports) Less(i, j int) bool {
	return o[i].Name < o[j].Name
}

func (o Imports) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

func (o Imports) String() string {
	os := make(map[string]Import)
	for _, imp := range o {
		os[imp.Path] = imp
	}
	var item = make(Imports, 0, len(os))
	for _, imp := range os {
		item = append(item, imp)
	}
	sort.Sort(item)
	imp := make([]string, 0, len(item))
	for _, value := range item {
		if value.HasAlias() {
			imp = append(imp, fmt.Sprintf("%s %q", value.Name, value.Path))
		} else {
			imp = append(imp, fmt.Sprintf("%q", value.Path))
		}
	}
	source := fmt.Sprintf("import (\n%s\n)", strings.Join(imp, "\n"))
	code, _ := format.Source([]byte(source))
	return string(code)
}
