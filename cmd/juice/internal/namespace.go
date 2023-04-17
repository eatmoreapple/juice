package internal

import (
	"bufio"
	"errors"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var errFound = errors.New("file found")

type NameSpaceAutoComplete struct {
	TypeName string
	_        struct{}
}

func (n NameSpaceAutoComplete) Autocomplete() (string, error) {
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

func (n NameSpaceAutoComplete) autoComplete(path string) (string, error) {
	var gomodPath = path
	for {
		ok, err := fileExists(filepath.Join(gomodPath, "go.mod"))
		if err != nil {
			return "", err
		}
		if ok {
			break
		}
		gomodPath = filepath.Dir(gomodPath)
	}
	// read go.mod and get module name
	f, err := os.Open(filepath.Join(gomodPath, "go.mod"))
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	var module string
	reader := bufio.NewReader(f)
	for {
		line, _, err := reader.ReadLine()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		data := string(line)
		if strings.HasPrefix(data, "module") {
			module = strings.TrimSpace(strings.TrimPrefix(data, "module"))
			break
		}
	}
	if module == "" {
		return "", errors.New("can not find module name")
	}
	var namespace string
	// find package name
	err = filepath.Walk(gomodPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filePath == gomodPath {
			return nil
		}
		if info.IsDir() {
			if info.Name() == "vendor" || !strings.HasPrefix(path, filePath) || strings.HasPrefix(path, ".") {
				return filepath.SkipDir
			}
		} else {
			if filepath.Dir(filePath) == path {
				relativePath, err := filepath.Rel(gomodPath, path)
				if err != nil {
					return err
				}
				if relativePath == "." {
					relativePath = ""
				}
				if relativePath != "" {
					relativePath = relativePath + "/"
				}
				pkgName := module + "/" + relativePath
				pkgName = strings.ReplaceAll(pkgName, "/", ".")
				namespace = pkgName + n.TypeName
				return errFound
			}
		}
		return nil
	})
	if err != nil && !errors.Is(err, errFound) {
		return "", err
	}
	if namespace == "" {
		return "", errors.New("can not find package name")
	}
	return namespace, nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
