package main

import (
	"errors"
	"log"
	"os"

	"github.com/eatmoreapple/juice/juicecli/impl"
	"github.com/eatmoreapple/juice/juicecli/tell"
)

// Command defines a command which can be executed by juice.
type Command interface {
	// Name returns the name of the command.
	// The name is used in the command line.
	// For example, if the name is "generate", the command is executed by "juice generate".
	// The name must be unique.
	Name() string

	// Do execute the command.
	Do() error

	// Help returns the help message of the command.
	Help() string
}

// cmdLibraries is a map of commands which can be executed by juice.
var cmdLibraries = make(map[string]Command)

// Register registers a command.
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

// Do execute the command.
func Do() error {
	if len(os.Args) < 2 {
		return errors.New("juice: command is required")
	}
	name := os.Args[1]
	switch name {
	case "--help":
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
	if err := Register(&tell.Generate{}); err != nil {
		log.Fatal(err)
	}
}

func main() {
	if err := Do(); err != nil {
		log.Println(err)
	}
}
