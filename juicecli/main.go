package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/eatmoreapple/juice/juicecli/impl"
	"github.com/eatmoreapple/juice/juicecli/internal/colorformat"
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

	// Description returns the description of the command.
	Description() string
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
	switch {
	case len(os.Args) == 1 || os.Args[1] == "--help" || os.Args[1] == "-h":
		fmt.Printf("command line tool for generating code.\n\n")
		fmt.Printf("Usage:\n")
		fmt.Printf("  juice [command]\n\n")
		fmt.Printf("Available Commands:\n")
		for _, cmd := range commands {
			fmt.Println(fmt.Sprintf("  %-10s %s", colorformat.Red(cmd.Name()), colorformat.Magenta(cmd.Description())))
		}
		fmt.Printf("\nFlags:\n")
		fmt.Println(colorformat.Cyan("  -h, --help\t"))
		fmt.Println("\nUse \"juice [command] --help\" for more information about a command.")
		return nil
	}
	name := os.Args[1]
	cmd, ok := cmdLibraries[name]
	if !ok {
		return errors.New("juice: unknown command " + name)
	}
	if len(os.Args) > 2 && (os.Args[2] == "--help" || os.Args[2] == "-h") {
		println(cmd.Help())
		return nil
	}
	return cmd.Do()
}

var commands = []Command{
	&impl.Generate{},
	&tell.Generate{},
}

func init() {
	for _, cmd := range commands {
		if err := Register(cmd); err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	if err := Do(); err != nil {
		log.Println(err)
	}
}
