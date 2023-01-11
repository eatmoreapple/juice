package scanner

type Generator struct {
	resultMap string
	typeName  string
}

func (g *Generator) Name() string {
	return "scanner"
}

func (g *Generator) Do() error {
	return nil
}
