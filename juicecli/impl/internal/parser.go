package internal

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	stdfs "io/fs"
	"os"
	"path/filepath"
	"strings"
	_ "unsafe"

	"github.com/eatmoreapple/juice"
	"github.com/eatmoreapple/juice/juicecli/internal/module"
	"github.com/eatmoreapple/juice/juicecli/internal/namespace"
)

//go:linkname newXMLConfigurationParser github.com/eatmoreapple/juice.newXMLConfigurationParser
func newXMLConfigurationParser(stdfs.FS, string, bool) (*juice.Configuration, error)

// CommandParser defines the interface for parsing command.
type CommandParser interface {
	RegisterCommand(cmd *flag.FlagSet)
	Parse() error
}

// CommandParserGroup wraps multiple CommandParser.
type CommandParserGroup struct {
	CommandParsers []CommandParser
	cmd            *flag.FlagSet
}

func (c *CommandParserGroup) RegisterCommand(cmd *flag.FlagSet) {
	for _, p := range c.CommandParsers {
		p.RegisterCommand(cmd)
	}
	c.cmd = cmd
}

func (c *CommandParserGroup) Parse() error {
	if err := c.cmd.Parse(os.Args[2:]); err != nil {
		return err
	}
	for _, p := range c.CommandParsers {
		if err := p.Parse(); err != nil {
			return err
		}
	}
	return nil
}

func NewCommandParserGroup(group []CommandParser) CommandParser {
	return &CommandParserGroup{CommandParsers: group}
}

type TypeNameParser struct {
	point *string
}

func (t *TypeNameParser) RegisterCommand(cmd *flag.FlagSet) {
	cmd.StringVar(t.point, "type", "", "typeName type name")
}

func (t *TypeNameParser) Parse() error {
	if *t.point == "" {
		return errors.New("typeName type name is required")
	}
	return nil
}

type ImplNameParser struct {
	point *string
}

func (t *ImplNameParser) RegisterCommand(cmd *flag.FlagSet) {
	cmd.StringVar(t.point, "impl", "", "implementation name")
}

func (t *ImplNameParser) Parse() error {
	return nil
}

// defaultConfigFiles is the default config file name
// while config is not set, we will check if config.xml or config/config.xml exists
var defaultConfigFiles = [...]string{"config.xml", "config/config.xml"}

type ConfigPathParser struct {
	point *string
}

func (t *ConfigPathParser) RegisterCommand(cmd *flag.FlagSet) {
	cmd.StringVar(t.point, "config", "", "config path, default: config.xml or config/config.xml")
}

func (t *ConfigPathParser) Parse() error {
	if *t.point != "" {
		// if config is set, it is not our responsibility to check if it exists
		// configparser will check it
		return nil
	}
	// if config is not set, we will check if config.xml or config/config.xml exists
	for _, f := range defaultConfigFiles {
		ok, err := fileExists(f)
		if err != nil {
			return err
		}
		if ok {
			*t.point = f
			return nil
		}
	}
	return errors.New("config.xml or config/config.xml not found")
}

type NamespaceParser struct {
	point    *string
	typeName *string
}

func (t *NamespaceParser) RegisterCommand(cmd *flag.FlagSet) {
	cmd.StringVar(t.point, "namespace", "", "namespace, default: auto generate")
}

func (t *NamespaceParser) Parse() error {
	if *t.point == "" {
		var err error
		cmp := &namespace.AutoComplete{TypeName: *t.typeName}
		*t.point, err = cmp.Autocomplete()
		if err != nil {
			return err
		}
	}
	return nil
}

type OutputPathParser struct {
	point *string
}

func (t *OutputPathParser) RegisterCommand(cmd *flag.FlagSet) {
	cmd.StringVar(t.point, "output", "", "output path")
}

func (t *OutputPathParser) Parse() error {
	if *t.point != "" {
		if *t.point != filepath.Base(strings.TrimPrefix(*t.point, "./")) {
			return errors.New("output path only support file name")
		}
	}
	return nil
}

type Parser struct {
	typeName  string
	impl      string
	cfg       string
	namespace string
	output    string
}

func (p *Parser) parseCommand() error {
	cmd := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	var parsers = []CommandParser{
		// TypeNameParser must be the first one
		// because it will set the default value for other parsers
		&TypeNameParser{point: &p.typeName},
		&ImplNameParser{point: &p.impl},
		&ConfigPathParser{point: &p.cfg},
		&NamespaceParser{point: &p.namespace, typeName: &p.typeName},
		&OutputPathParser{point: &p.output},
	}
	ps := NewCommandParserGroup(parsers)
	ps.RegisterCommand(cmd)
	return ps.Parse()
}

func (p *Parser) Parse() (*Generator, error) {
	if err := p.parseCommand(); err != nil {
		return nil, err
	}
	return p.parse()
}

func (p *Parser) parse() (*Generator, error) {
	cfg, err := newXMLConfigurationParser(juice.LocalFS{}, p.cfg, true)
	if err != nil {
		return nil, err
	}

	// set default impl name
	if p.impl == "" {
		implSuffix := cfg.Settings.Get("implSuffix")
		if implSuffix == "" {
			implSuffix = "Impl"
		}
		p.impl = p.typeName + implSuffix.String()
	}

	// find type node
	node, file, err := module.FindTypeNode("./", p.typeName)
	if err != nil {
		return nil, err
	}
	iface, ok := node.(*ast.InterfaceType)
	if !ok {
		return nil, fmt.Errorf("%s is not an interface", p.typeName)
	}
	impl := newImplement(file, p.typeName, p.impl)
	if err = impl.Init(iface); err != nil {
		return nil, err
	}
	return &Generator{
		cfg:       cfg,
		impl:      impl,
		namespace: p.namespace,
		output:    p.output,
	}, nil
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
