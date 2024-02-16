package namespace

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/eatmoreapple/juice/juicecli/internal/module"
)

type AutoComplete struct {
	TypeName string
	_        struct{}
}

func (n AutoComplete) Autocomplete() (string, error) {
	path, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// is current package main?
	fs := token.NewFileSet()
	pkg, err := parser.ParseDir(fs, path, nil, parser.PackageClauseOnly)
	if err != nil {
		return "", err
	}
	if len(pkg) == 1 {
		for _, v := range pkg {
			if v.Name == "main" {
				return "main." + n.TypeName, nil
			}
		}
	}
	return n.autoComplete(path)
}

func (n AutoComplete) autoComplete(path string) (string, error) {
	goModPath, err := module.FindGoModPath(path)
	if err != nil {
		return "", err
	}
	// read go.mod and get module name
	f, err := os.Open(filepath.Join(goModPath, "go.mod"))
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	pkg, err := module.ParseGoModuleName(f)
	if err != nil {
		return "", err
	}
	// find package name
	relativePath, err := filepath.Rel(goModPath, path)
	if err != nil {
		return "", err
	}
	if relativePath == "." {
		relativePath = ""
	}
	if relativePath != "" {
		relativePath = relativePath + "/"
	}
	pkgName := pkg + "/" + relativePath
	namespace := pkgName + n.TypeName
	replacer := strings.NewReplacer("/", ".", "\\", ".")
	return replacer.Replace(namespace), nil
}
