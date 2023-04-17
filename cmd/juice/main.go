package main

import (
	"errors"
	"log"
	"os"

	"github.com/eatmoreapple/juice/cmd/juice/impl"
	_ "github.com/eatmoreapple/juice/cmd/juice/impl"
	"github.com/eatmoreapple/juice/cmd/juice/namespace"
	_ "github.com/eatmoreapple/juice/cmd/juice/namespace"
)

type Command interface {
	Name() string
	Do() error
	Help() string
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
		return errors.New("juice: command is required")
	}
	name := os.Args[1]
	if name == "--help" {
		println("juice is a command line tool for generating code.")
		println("  Usage: juice command [options] [arguments]")
		println("  Options:")
		println("    --help")
		println("      show help")
		println("  Commands:")
		for _, cmd := range cmdLibraries {
			println("    " + cmd.Name())
		}
		return nil
	}
	cmd, ok := cmdLibraries[name]
	if !ok {
		return errors.New("juice: unknown command " + name)
	}
	if len(os.Args) > 2 && os.Args[2] == "--help" {
		println(cmd.Help())
		return nil
	}
	return cmd.Do()
}

func init() {
	if err := Register(&impl.Generate{}); err != nil {
		log.Fatal(err)
	}
	if err := Register(&namespace.Generate{}); err != nil {
		log.Fatal(err)
	}
}

func main() {
	if err := Do(); err != nil {
		log.Println(err)
	}
}
