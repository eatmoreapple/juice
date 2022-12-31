package cmd

import (
	"fmt"
	"strings"
)

type Implement struct {
	// Package is a package of implement.
	Package string
	// Name is a name of implement.
	Name string
	// Interface is an interface of implement.
	Interface string
	// Methods is a methods of implement.
	Methods Functions
	// ExtraImports is extra imports of implement.
	ExtraImports Imports
}

func (i Implement) Imports() Imports {
	ps := make(map[string]Import)
	for _, method := range i.Methods {
		for _, imp := range method.Args.Imports() {
			ps[imp.Path] = imp
		}
		for _, imp := range method.Results.Imports() {
			ps[imp.Path] = imp
		}
		if method.Receiver != nil && method.Receiver.Import.Path != "" {
			ps[method.Receiver.Import.Path] = method.Receiver.Import
		}
		for _, imp := range i.ExtraImports {
			ps[imp.Path] = imp
		}
	}
	var result = make(Imports, 0, len(ps))
	for _, imp := range ps {
		result = append(result, imp)
	}
	return result
}

func (i Implement) String() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("package %s", i.Package))
	builder.WriteString("\n\n")
	builder.WriteString(i.Imports().String())
	builder.WriteString("\n\n")
	builder.WriteString(fmt.Sprintf("type %s struct {}", i.Name))
	builder.WriteString("\n\n")
	for index, method := range i.Methods {
		builder.WriteString(method.String())
		if index < len(i.Methods)-1 {
			builder.WriteString("\n\n")
		}
	}
	builder.WriteString("\n\n")
	builder.WriteString(i.constructor().String())
	return formatCode(builder.String())
}

func (i Implement) constructor() Function {
	var body = fmt.Sprintf("\n\treturn &%s{}", i.Name)
	return Function{
		Name: fmt.Sprintf("New%s", i.Interface),
		Results: Values{
			{Type: i.Interface},
		},
		Body: &body,
	}
}
