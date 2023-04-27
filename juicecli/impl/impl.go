package impl

import (
	"fmt"
	"strings"

	"github.com/eatmoreapple/juice/juicecli/impl/internal"
	"github.com/eatmoreapple/juice/juicecli/internal/colorformat"
)

// Generate is a command for generating implementation.
type Generate struct{}

func (i *Generate) Name() string {
	return "impl"
}

func (i *Generate) Do() error {
	parser := internal.Parser{}
	return parser.Parse()
}

func (i *Generate) Help() string {
	var builder strings.Builder
	builder.WriteString("command for generating implementation.\n\n")
	builder.WriteString("Usage: juicecli impl [options] [arguments] \n\n")
	builder.WriteString("Options:\n")
	builder.WriteString(fmt.Sprintf("    %s %s\n", colorformat.Red("--type"), colorformat.Magenta("string")))
	builder.WriteString(colorformat.Green("      implementation type name.\n\n"))
	builder.WriteString(fmt.Sprintf("    %s %s\n", colorformat.Red("--output"), colorformat.Magenta("string")))
	builder.WriteString(colorformat.Green("      output file name, default is stdout.\n\n"))
	builder.WriteString(fmt.Sprintf("    %s %s\n", colorformat.Red("--config"), colorformat.Magenta("string")))
	builder.WriteString(colorformat.Green("      config file path, default is \"config.xml\" OR \"config/config.xml\".\n\n"))
	builder.WriteString(fmt.Sprintf("    %s %s\n", colorformat.Red("--namespace"), colorformat.Magenta("string")))
	builder.WriteString(colorformat.Green("      namespace, default is auto generate.\n\n"))
	builder.WriteString(fmt.Sprintf("    %s %s\n", colorformat.Red("--impl"), colorformat.Magenta("string")))
	builder.WriteString(colorformat.Green("      implementation name, default is auto generate."))
	return builder.String()
}

func (i *Generate) Description() string {
	return "impl is a command for generating implementation."
}
