package ast

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestInterface(t *testing.T) {
	var src = `
package internal

import (
	"context"
	"go/token"
	p "go/parser"
)

type Interface interface {
	// GetUserByID 根据用户id查找用户
	GetUserByID(context.Context, *token.FileSet, p.Mode) (int64, error)
}

type User struct{}

`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.TypeSpec:
			kind, ok := n.Type.(*ast.InterfaceType)
			if !ok {
				return true
			}
			iface := &Interface{kind}
			for _, m := range iface.Methods() {
				t.Log(m.Signature())
				t.Log(m.Imports(f.Imports))
			}
		}
		return true
	})
}
