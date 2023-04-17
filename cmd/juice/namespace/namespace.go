package namespace

import (
	"errors"
	"flag"
	"os"

	"github.com/eatmoreapple/juice/cmd/juice/internal"
	"github.com/eatmoreapple/juice/cmd/juice/internal/cmd"
)

type Namespace struct{}

func (n *Namespace) Name() string {
	return "namespace"
}

func (n *Namespace) Do() error {
	var _type string
	c := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	c.StringVar(&_type, "type", "", "typeName type name")
	_ = c.Parse(os.Args[2:])
	if _type == "" {
		return errors.New("namespace: type is required")
	}
	cmp := &internal.NameSpaceAutoComplete{TypeName: _type}
	namespace, err := cmp.Autocomplete()
	if err != nil {
		return err
	}
	println(namespace)
	return nil
}

func init() {
	if err := cmd.Register(&Namespace{}); err != nil {
		panic(err)
	}
}
