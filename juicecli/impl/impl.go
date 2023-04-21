package impl

import (
	"strings"

	"github.com/eatmoreapple/juice/juicecli/impl/internal"
)

// Generate is a command for generating implementation.
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

func (i *Generate) Help() string {
	var builder strings.Builder
	builder.WriteString("impl is a command for generating implementation.\n")
	builder.WriteString("  Usage: juice impl [options] [arguments] \n")
	builder.WriteString("  Options:\n")
	builder.WriteString("    --type string\n")
	builder.WriteString("      typeName type name\n")
	builder.WriteString("    --output string\n")
	builder.WriteString("      output file name, default is stdout\n")
	builder.WriteString("    --config string\n")
	builder.WriteString("      config file path, default is ./config.xml OR ./config/config.xml\n")
	builder.WriteString("    --namespace string\n")
	builder.WriteString("      namespace, default is auto generate\n")
	builder.WriteString("    -impl string\n")
	builder.WriteString("      implementation name, default is auto generate")
	return builder.String()
}
