package ast

import (
	"fmt"
	"go/ast"
	"log"
	"strings"
)

const (
	ParamPrefix  = "arg"
	ResultPrefix = "result"
)

func isBuiltInType(name string) bool {
	switch name {
	case "int", "int8", "int16", "int32", "int64":
		return true
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return true
	case "float32", "float64":
		return true
	case "string":
		return true
	case "bool":
		return true
	case "complex64", "complex128":
		return true
	default:
		return false
	}
}

// Value is a value of interface, which wraps ast.Field.
type Value struct {
	*ast.Field
	name string
}

// TypeName returns the type name of value.
// If the value is a pointer, the type name will be prefixed with "*".
// If the value is a slice, the type name will be prefixed with "[]".
func (v *Value) TypeName() string {
	switch t := v.Type.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		value := &Value{Field: &ast.Field{Type: t.X}}
		return "*" + value.TypeName()
	case *ast.ArrayType:
		value := &Value{Field: &ast.Field{Type: t.Elt}}
		return "[]" + value.TypeName()
	case *ast.MapType:
		key := &Value{Field: &ast.Field{Type: t.Key}}
		if key.TypeName() != "string" {
			log.Fatalf("map key must be string, but got %s", key.TypeName())
		}
		value := &Value{Field: &ast.Field{Type: t.Value}}
		return "map[" + key.TypeName() + "]" + value.TypeName()
	case *ast.SelectorExpr:
		if name := v.ImportPackageName(); name != "" {
			return name + "." + t.Sel.Name
		}
		return t.Sel.Name
	default:
		log.Fatal("unknown type")
		return ""
	}
}

// String returns the string representation of value.
func (v *Value) String(prefix string, index int) string {
	if name := v.Name(); name != "" {
		return name + " " + v.TypeName()
	} else {
		return fmt.Sprintf("%s%d %s", prefix, index, v.TypeName())
	}
}

// Name returns the name of the value.
func (v *Value) Name() string {
	return v.name
}

// ImportPackageName returns the package name of the import.
func (v *Value) ImportPackageName() string {
	return importTypeName(v.Type)
}

func (v *Value) IsBuiltInType() bool {
	return isBuiltInType(v.DirectTypename())
}

func (v *Value) IsPointerType() bool {
	return strings.HasPrefix(v.TypeName(), "*")
}

func (v *Value) DirectTypename() string {
	return strings.TrimLeft(v.TypeName(), "*")
}

// ValueGroup is a group of Value. It is used to represent the return values of a method.
type ValueGroup []*Value

func (vs ValueGroup) Imports(pkgImports []*ast.ImportSpec) ImportGroup {
	var result ImportGroup
	for _, v := range vs {
		if name := v.ImportPackageName(); name != "" {
			if imp := findImport(name, pkgImports); imp != nil {
				result = append(result, &Import{ImportSpec: imp})
			}
		}
	}
	return result
}

func (vs ValueGroup) String(prefix string) string {
	var builder strings.Builder
	if len(vs) == 0 {
		return ""
	}
	for i, v := range vs {
		builder.WriteString(v.String(prefix, i))
		if i < len(vs)-1 {
			builder.WriteString(", ")
		}
	}
	return builder.String()
}

func (vs ValueGroup) NameAt(prefix string, index int) string {
	name := vs[index].Name()
	if name == "" {
		return fmt.Sprintf("%s%d", prefix, index)
	}
	return name
}

// valueGroupFrom returns a ValueGroup from fields.
func valueGroupFrom(fields []*ast.Field) ValueGroup {
	var result = make(ValueGroup, 0, len(fields))
	for _, param := range fields {
		if len(param.Names) == 0 {
			result = append(result, &Value{Field: param})
			continue
		}
		for _, name := range param.Names {
			result = append(result, &Value{Field: param, name: name.Name})
		}
	}
	return result
}

type Interface struct{ *ast.InterfaceType }

// Methods returns all methods of interface.
func (i *Interface) Methods() []*Function {
	var result = make([]*Function, 0, len(i.InterfaceType.Methods.List))
	for _, method := range i.InterfaceType.Methods.List {
		result = append(result, &Function{Field: method})
	}
	return result
}

// Imports returns all imports of interface.
func (i *Interface) Imports(pkgImports []*ast.ImportSpec) ImportGroup {
	var result ImportGroup
	for _, method := range i.Methods() {
		result = append(result, method.Imports(pkgImports)...)
	}
	return result.Uniq()
}

// Function wraps ast.Field to provide some useful methods.
type Function struct{ *ast.Field }

// Name returns the name of the function.
func (f *Function) Name() string {
	if len(f.Names) == 0 {
		return ""
	}
	return f.Names[0].Name
}

func (f *Function) Comment() string {
	if f.Doc == nil {
		return ""
	}
	var builder strings.Builder
	for _, comment := range f.Doc.List {
		builder.WriteString(comment.Text)
		builder.WriteString("\n")
	}
	return builder.String()
}

// Signature returns the signature of function.
func (f *Function) Signature() string {
	var builder strings.Builder
	builder.WriteString(f.Name())
	builder.WriteString("(")
	builder.WriteString(f.Params().String(ParamPrefix))
	builder.WriteString(") ")
	if result := f.Results().String(ResultPrefix); result != "" {
		builder.WriteString("(")
		builder.WriteString(result)
		builder.WriteString(")")
	}
	return builder.String()
}

// Params returns all params of function.
func (f *Function) Params() ValueGroup {
	method, ok := f.Type.(*ast.FuncType)
	if !ok {
		return nil
	}
	return valueGroupFrom(method.Params.List)
}

// Results returns all results of function.
func (f *Function) Results() ValueGroup {
	method, ok := f.Type.(*ast.FuncType)
	if !ok {
		return nil
	}
	return valueGroupFrom(method.Results.List)
}

// Imports returns all imports of function.
func (f *Function) Imports(pkgImports []*ast.ImportSpec) ImportGroup {
	return append(f.Params().Imports(pkgImports), f.Results().Imports(pkgImports)...).Uniq()
}

func importTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return importTypeName(t.X)
	case *ast.SelectorExpr:
		return importTypeName(t.X)
	case *ast.ArrayType:
		return importTypeName(t.Elt)
	case *ast.MapType:
		return importTypeName(t.Value)
	case *ast.Ident:
		return t.Name
	}
	return ""
}
