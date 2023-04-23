package internal

import (
	"fmt"
	"go/ast"
	"strings"
)

type Implement struct {
	// Name is a name of implement.
	Name string
	// Interface is an interface of implement.
	Interface string
	// methods is a methods of implement.
	methods Functions
	// ExtraImports is extra imports of implement.
	ExtraImports Imports
	//
	file *ast.File
}

func (i *Implement) Imports() Imports {
	ps := make(map[string]Import)
	for _, method := range i.methods {
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

func (i *Implement) Package() string {
	return i.file.Name.Name
}

func (i *Implement) String() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("package %s", i.Package()))
	builder.WriteString("\n\n")
	builder.WriteString(i.Imports().String())
	builder.WriteString("\n\n")
	builder.WriteString(fmt.Sprintf("type %s struct {}", i.Name))
	builder.WriteString("\n\n")
	for index, method := range i.methods {
		builder.WriteString(method.String())
		if index < len(i.methods)-1 {
			builder.WriteString("\n\n")
		}
	}
	builder.WriteString("\n\n")
	constructor := i.constructor()
	builder.WriteString(fmt.Sprintf("// %s returns a new %s.\n", constructor.Name, i.Interface))
	builder.WriteString(constructor.String())
	return formatCode(builder.String())
}

func (i *Implement) constructor() Function {
	var body = fmt.Sprintf("\n\treturn &%s{}", i.Name)
	return Function{
		Name: fmt.Sprintf("New%s", i.Interface),
		Results: Values{
			{Type: i.Interface},
		},
		Body: &body,
	}
}

func (i *Implement) Init(iface *ast.InterfaceType) error {
	for _, method := range iface.Methods.List {
		if len(method.Names) == 0 {
			continue
		}
		methodName := method.Names[0].Name
		ft, ok := method.Type.(*ast.FuncType)
		if !ok {
			return fmt.Errorf("method %s is not a function type", methodName)
		}

		argsValue := parseValues(i.file, ft.Params.List)
		returnValues := parseValues(i.file, ft.Results.List)

		function := &Function{
			Name:    methodName,
			Args:    argsValue,
			Results: returnValues,
			Receiver: &Value{
				Type: i.Name,
				Name: strings.ToLower(i.Name[:1]),
			},
			Type: i.Interface,
		}

		if method.Doc != nil {
			var builder strings.Builder
			for _, doc := range method.Doc.List {
				builder.WriteString(doc.Text)
				builder.WriteString("\n")
			}
			text := builder.String()
			function.Doc = &text
		}
		i.methods = append(i.methods, function)
	}
	return nil
}

func newImplement(file *ast.File, input, output string) *Implement {
	impl := &Implement{
		Name:      output,
		Interface: input,
		file:      file,
	}
	return impl
}
