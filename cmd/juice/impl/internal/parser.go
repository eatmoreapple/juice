package internal

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/eatmoreapple/juice"
)

type Parser struct {
	typeName  string
	impl      string
	cfg       string
	path      string
	namespace string
	output    string
}

func (p *Parser) parseCommand() error {
	cmd := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	cmd.StringVar(&p.typeName, "type", "", "typeName type name")
	cmd.StringVar(&p.impl, "impl", "", "implementation name")
	cmd.StringVar(&p.cfg, "config", "", "config path")
	cmd.StringVar(&p.path, "path", "./", "path")
	cmd.StringVar(&p.namespace, "namespace", "", "namespace")
	cmd.StringVar(&p.output, "output", "", "output path")
	return cmd.Parse(os.Args[2:])
}

func (p *Parser) Parse() (*Generator, error) {
	if err := p.parseCommand(); err != nil {
		return nil, err
	}
	if p.typeName == "" {
		return nil, errors.New("typeName type name is required")
	}
	if p.namespace == "" {
		return nil, errors.New("namespace is required")
	}
	if p.impl == "" {
		p.impl = p.typeName + "Impl"
	}
	if p.output != "" {
		if p.output != filepath.Base(strings.TrimPrefix(p.output, "./")) {
			return nil, errors.New("output path only support file name")
		}
	}
	return p.parse()
}

func (p *Parser) parse() (*Generator, error) {
	cfg, err := juice.NewXMLConfiguration(p.cfg)
	if err != nil {
		return nil, err
	}
	pkgs, err := parser.ParseDir(token.NewFileSet(), p.path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			impl, err := inspect(f, p.typeName, p.impl)
			if err != nil {
				return nil, err
			}
			if impl != nil {
				return &Generator{
					cfg:       cfg,
					impl:      impl,
					namespace: p.namespace,
					output:    p.output,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("type %s not found", p.typeName)
}

func inspect(node ast.Node, input, output string) (*Implement, error) {
	var (
		impl  Implement
		err   error
		found bool
	)
	impl.Name = output
	impl.Interface = input
	f := node.(*ast.File)
	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}
		switch x := n.(type) {
		case *ast.TypeSpec:
			if x.Name.Name == input {
				typ, ok := x.Type.(*ast.InterfaceType)
				if !ok {
					err = fmt.Errorf("type %s is not an interface", input)
					return true
				}
				impl.Package = f.Name.Name
				for _, method := range typ.Methods.List {
					if len(method.Names) == 0 {
						continue
					}
					methodName := method.Names[0].Name

					argsValue := parseValues(f, method.Type.(*ast.FuncType).Params.List)

					returnValues := parseValues(f, method.Type.(*ast.FuncType).Results.List)

					function := &Function{
						Name:    methodName,
						Args:    argsValue,
						Results: returnValues,
						Receiver: &Value{
							Type: output,
							Name: strings.ToLower(output[:1]),
						},
						Type: input,
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

					impl.Methods = append(impl.Methods, function)
				}
				found = true
				return false
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &impl, nil
}

func parseValues(file *ast.File, fields []*ast.Field) Values {
	var values Values
	for _, field := range fields {
		value := Value{}
		parseValue(&value, file, field)
		if len(field.Names) > 0 {
			value.Name = field.Names[0].Name
		}
		values = append(values, value)
	}
	return values
}

func parseValue(value *Value, file *ast.File, field *ast.Field) {
	switch t := field.Type.(type) {
	case *ast.Ident:
		value.Type = t.Name
	case *ast.SelectorExpr:
		value.Type = t.Sel.Name
		parseImport(value, file, t.X.(*ast.Ident).Name)
	case *ast.ArrayType:
		value.IsSlice = true
		parseValue(value, file, &ast.Field{Type: t.Elt})
	case *ast.StarExpr:
		value.IsPointer = true
		parseValue(value, file, &ast.Field{Type: t.X})
	case *ast.MapType:
		value.IsMap = true
		if t.Key.(*ast.Ident).Name != "string" {
			panic("map key must be string")
		}
		parseValue(value, file, &ast.Field{Type: t.Value})
	case *ast.InterfaceType:
		value.Type = "interface{}"
	}
}

func parseImport(value *Value, file *ast.File, alias string) {
	for _, spec := range file.Imports {
		pkgName := strings.Trim(spec.Path.Value, `"`)
		if spec.Name != nil && spec.Name.Name == alias {
			value.Import.Path = pkgName
			value.Import.Name = alias
			break
		}
		pkg := strings.Split(pkgName, "/")
		if pkg[len(pkg)-1] == alias {
			value.Import.Path = pkgName
			value.Import.Name = alias
			break
		}
	}
}
