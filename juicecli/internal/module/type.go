package module

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

func FindTypeNode(path, typeName string) (node ast.Node, file *ast.File, err error) {
	pkgs, err := parser.ParseDir(token.NewFileSet(), path, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			if node != nil {
				return
			}
			ast.Inspect(f, func(n ast.Node) bool {
				if node != nil {
					return false
				}
				switch x := n.(type) {
				case *ast.TypeSpec:
					if x.Name.Name == typeName {
						node = x.Type
						file = f
						return false
					}
				}
				return true
			})
		}
	}
	if node == nil {
		err = fmt.Errorf("type %s not found", typeName)
	}
	return
}
