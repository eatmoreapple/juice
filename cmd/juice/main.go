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
	if cmd, ok := cmdLibraries[name]; ok {
		return cmd.Do()
	}
	return errors.New("juice: unknown command " + name)
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
