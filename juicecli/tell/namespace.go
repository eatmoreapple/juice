package tell

import (
	"errors"
	"flag"
	"os"
	"strings"

	"github.com/eatmoreapple/juice/juicecli/internal/module"
	"github.com/eatmoreapple/juice/juicecli/internal/namespace"
)

// Generate is a command for generating namespace.
type Generate struct {
	typeName string
	check    bool
}

func (n *Generate) Name() string {
	return "tell"
}

func (n *Generate) Do() error {
	c := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	c.StringVar(&n.typeName, "type", "", "typeName type name")
	c.BoolVar(&n.check, "check", true, "check if type is exists")

	if err := c.Parse(os.Args[2:]); err != nil {
		return err
	}

	if n.typeName == "" {
		return errors.New("namespace: type is required")
	}
	if n.check {
		if _, _, err := module.FindTypeNode(".", n.typeName); err != nil {
			return err
		}
	}

	cmp := &namespace.AutoComplete{TypeName: n.typeName}
	data, err := cmp.Autocomplete()
	if err != nil {
		return err
	}
	println(data)
	return nil
}

func (n *Generate) Help() string {
	var builder strings.Builder
	builder.WriteString("namespace: generate namespace for type\n")
	builder.WriteString("  Usage:\n")
	builder.WriteString("    --type string interface type name\n")
	return builder.String()
}
