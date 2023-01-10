package cmd

import (
	"errors"
	"os"

	"github.com/eatmoreapple/juice/cmd/internal"
)

type Command interface {
	Name() string
	Do() error
}

type ImplGenerate struct{}

func (i *ImplGenerate) Name() string {
	return "impl"
}

func (i *ImplGenerate) Do() error {
	parser := internal.Parser{}
	impl, err := parser.Parse()
	if err != nil {
		return err
	}
	return impl.Generate()
}

var cmdLibraries = make(map[string]Command)

func Register(cmd Command) error {
	if cmd == nil {
		return errors.New("cmd is nil")
	}
	if _, ok := cmdLibraries[cmd.Name()]; ok {
		return errors.New("cmd: duplicate command " + cmd.Name())
	}
	cmdLibraries[cmd.Name()] = cmd
	return nil
}

func Do() error {
	if len(os.Args) < 2 {
		return errors.New("cmd: command is required")
	}
	name := os.Args[1]
	if cmd, ok := cmdLibraries[name]; ok {
		return cmd.Do()
	}
	return errors.New("cmd: unknown command " + name)
}

func init() {
	_ = Register(&ImplGenerate{})
}
