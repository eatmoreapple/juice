package internal

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"io"
	"os"
	"path/filepath"
	"strings"
	_ "unsafe"

	"github.com/eatmoreapple/juice"
	"github.com/eatmoreapple/juice/juicecli/internal/module"
	"github.com/eatmoreapple/juice/juicecli/internal/namespace"
)

//go:linkname newLocalXMLConfiguration github.com/eatmoreapple/juice.newLocalXMLConfiguration
func newLocalXMLConfiguration(string, bool) (*juice.Configuration, error)

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

// RegisterCommand registers all CommandParser.
func (c *CommandParserGroup) RegisterCommand(cmd *flag.FlagSet) {
	for _, p := range c.CommandParsers {
		p.RegisterCommand(cmd)
	}
	c.cmd = cmd
}

// Parse parses all CommandParser.
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

// NewCommandParserGroup creates a new CommandParserGroup which wraps multiple CommandParser.
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

func (p *Parser) Parse() error {
	if err := p.parseCommand(); err != nil {
		return err
	}
	return p.parse()
}

func (p *Parser) parse() error {
	cfg, err := newLocalXMLConfiguration(p.cfg, true)
	if err != nil {
		return err
	}

	// set default impl name
	if p.impl == "" {
		implSuffix := cfg.Settings().Get("implSuffix")
		if implSuffix == "" {
			implSuffix = "Impl"
		}
		p.impl = p.typeName + implSuffix.String()
	}

	// find type node
	node, file, err := module.FindTypeNode("./", p.typeName)
	if err != nil {
		return err
	}
	iface, ok := node.(*ast.InterfaceType)
	if !ok {
		return fmt.Errorf("%s is not an interface", p.typeName)
	}

	impl := newImplement(file, iface, p.typeName, p.impl)

	generator := newGenerator(p.namespace, cfg, impl)

	var output io.Writer = os.Stdout
	if p.output != "" {
		output, err = os.Create(p.output)
		if err != nil {
			return err
		}
		defer func() { _ = output.(io.Closer).Close() }()
	}
	_, err = generator.WriteTo(output)
	return err
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
