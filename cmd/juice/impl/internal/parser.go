package internal

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	stdfs "io/fs"
	"os"
	"path/filepath"
	"strings"
	_ "unsafe"

	"github.com/eatmoreapple/juice"
	"github.com/eatmoreapple/juice/cmd/juice/internal"
)

//go:linkname newXMLConfigurationParser github.com/eatmoreapple/juice.newXMLConfigurationParser
func newXMLConfigurationParser(stdfs.FS, string, bool) (*juice.Configuration, error)

type Parser struct {
	typeName  string
	impl      string
	cfg       string
	namespace string
	output    string
}

func (p *Parser) parseCommand() error {
	cmd := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	cmd.StringVar(&p.typeName, "type", "", "typeName type name")
	cmd.StringVar(&p.impl, "impl", "", "implementation name")
	cmd.StringVar(&p.cfg, "config", "", "config path")
	cmd.StringVar(&p.namespace, "namespace", "", "namespace")
	cmd.StringVar(&p.output, "output", "", "output path")
	return cmd.Parse(os.Args[2:])
}

func (p *Parser) init() error {
	if p.typeName == "" {
		return errors.New("typeName type name is required")
	}
	if p.namespace == "" {
		var err error
		cmp := &internal.NameSpaceAutoComplete{TypeName: p.typeName}
		p.namespace, err = cmp.Autocomplete()
		if err != nil {
			return err
		}
	}
	if p.output != "" {
		if p.output != filepath.Base(strings.TrimPrefix(p.output, "./")) {
			return errors.New("output path only support file name")
		}
	}
	if err := p.parseCfg(); err != nil {
		return err
	}
	return nil
}

var ErrConfigNotFound = errors.New("config.xml or config/config.xml not found")

// parseCfg parse config.xml or config/config.xml
func (p *Parser) parseCfg() error {
	if ok, err := fileExists("config.xml"); err != nil {
		return err
	} else if ok {
		p.cfg = "config.xml"
		return nil
	} else if ok, err := fileExists("config/config.xml"); err != nil {
		return err
	} else if ok {
		p.cfg = "config/config.xml"
		return nil
	}
	return ErrConfigNotFound
}

func (p *Parser) Parse() (*Generator, error) {
	if err := p.parseCommand(); err != nil {
		return nil, err
	}
	if err := p.init(); err != nil {
		return nil, err
	}
	return p.parse()
}

func (p *Parser) parse() (*Generator, error) {
	cfg, err := newXMLConfigurationParser(juice.LocalFS{}, p.cfg, true)
	if err != nil {
		return nil, err
	}
	pkgs, err := parser.ParseDir(token.NewFileSet(), "./", nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// get implementation name from config
	if p.impl == "" {
		implSuffix := cfg.Settings.Get("implSuffix")
		if implSuffix == "" {
			implSuffix = "Impl"
		}
		p.impl = p.typeName + implSuffix.String()
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

func inspect(node *ast.File, input, output string) (*Implement, error) {
	impl := &Implement{
		Name:      output,
		Interface: input,
	}
	var err error
	ast.Inspect(node, func(n ast.Node) bool {
		if impl.Package != "" {
			return false
		}
		switch x := n.(type) {
		case *ast.TypeSpec:
			if x.Name.Name != input {
				return true
			}
			typ, ok := x.Type.(*ast.InterfaceType)
			if !ok {
				err = fmt.Errorf("type %s is not an interface", input)
				return false
			}
			impl.Package = node.Name.Name
			for _, method := range typ.Methods.List {
				if len(method.Names) == 0 {
					continue
				}
				methodName := method.Names[0].Name
				ft, ok := method.Type.(*ast.FuncType)
				if !ok {
					err = fmt.Errorf("method %s is not a function type", methodName)
					return false
				}
				argsValue := parseValues(node, ft.Params.List)
				returnValues := parseValues(node, ft.Results.List)

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
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	if impl.Package == "" {
		return nil, nil
	}
	return impl, nil
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
		if ident, ok := t.X.(*ast.Ident); ok {
			parseImport(value, file, ident.Name)
		}
	case *ast.ArrayType:
		value.IsSlice = true
		parseValue(value, file, &ast.Field{Type: t.Elt})
	case *ast.StarExpr:
		value.IsPointer = true
		parseValue(value, file, &ast.Field{Type: t.X})
	case *ast.MapType:
		value.IsMap = true
		if ident, ok := t.Key.(*ast.Ident); ok && ident.Name != "string" {
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

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
