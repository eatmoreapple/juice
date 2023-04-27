package ast

import (
	"go/ast"
	"strings"
)

type Import struct{ *ast.ImportSpec }

func (i *Import) String() string {
	if i.Name != nil {
		return i.Name.Name + " " + i.Path.Value
	}
	return i.Path.Value
}

// Usage returns the name of the import.
// If the import has no name, it returns the last part of the path.
// For example
//
//		 "github.com/eatmoreapple/juice"      =>  juice
//	     "context"						   	  =>  context
//	     j "github.com/eatmoreapple/juice"    =>  j
func (i *Import) Usage() string {
	if i.Name != nil {
		return i.Name.Name
	}
	replace := strings.NewReplacer(`"`, "")
	text := strings.Split(replace.Replace(i.Path.Value), "/")
	return text[len(text)-1]
}

// ImportGroup is a group of imports.
type ImportGroup []*Import

// String returns the string representation of ImportGroup.
func (ig ImportGroup) String() string {
	if len(ig) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString("import (\n")
	for _, imp := range ig {
		builder.WriteString("\t")
		builder.WriteString(imp.String())
		builder.WriteString("\n")
	}
	builder.WriteString(")")
	return builder.String()
}

// Uniq returns a new ImportGroup with unique imports.
func (ig ImportGroup) Uniq() ImportGroup {
	var set = make(map[string]struct{})
	exists := func(imp *Import) bool {
		if _, ok := set[imp.Usage()]; ok {
			return true
		}
		set[imp.Usage()] = struct{}{}
		return false
	}
	var result = make(ImportGroup, 0, len(ig))
	for _, imp := range ig {
		if !exists(imp) {
			result = append(result, imp)
		}
	}
	return result
}

// findImport finds the import with the given name.
func findImport(name string, imports []*ast.ImportSpec) *ast.ImportSpec {
	for _, imp := range imports {
		each := &Import{ImportSpec: imp}
		if each.Usage() == name {
			return imp
		}
	}
	return nil
}
