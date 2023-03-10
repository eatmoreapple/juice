package impl

import (
	"github.com/eatmoreapple/juice/cmd/juice/impl/internal"
	"github.com/eatmoreapple/juice/cmd/juice/internal/cmd"
)

type ImplementGenerate struct{}

func (i *ImplementGenerate) Name() string {
	return "impl"
}

func (i *ImplementGenerate) Do() error {
	parser := internal.Parser{}
	impl, err := parser.Parse()
	if err != nil {
		return err
	}
	return impl.Generate()
}

func init() {
	_ = cmd.Register(&ImplementGenerate{})
}
