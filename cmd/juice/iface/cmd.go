package iface

import (
	"errors"
	"os"
)

type Command interface {
	Name() string
	Do() error
}

var cmdLibraries = make(map[string]Command)

func Register(cmd Command) error {
	if cmd == nil {
		return errors.New("iface is nil")
	}
	if _, ok := cmdLibraries[cmd.Name()]; ok {
		return errors.New("iface: duplicate command " + cmd.Name())
	}
	cmdLibraries[cmd.Name()] = cmd
	return nil
}

func Do() error {
	if len(os.Args) < 2 {
		return errors.New("iface: command is required")
	}
	name := os.Args[1]
	if cmd, ok := cmdLibraries[name]; ok {
		return cmd.Do()
	}
	return errors.New("iface: unknown command " + name)
}
