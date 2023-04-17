package impl

import (
	"github.com/eatmoreapple/juice/cmd/juice/impl/internal"
)

type Generate struct{}

func (i *Generate) Name() string {
	return "impl"
}

func (i *Generate) Do() error {
	parser := internal.Parser{}
	impl, err := parser.Parse()
	if err != nil {
		return err
	}
	return impl.Generate()
}
