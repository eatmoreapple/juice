package module

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ParseGoModuleName parse go.mod file and return module name
func ParseGoModuleName(f io.Reader) (string, error) {
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
	return module, nil
}

// FindGoModPath go.mod file and return path of go.mod
func FindGoModPath(path string) (string, error) {
	var goModPath = path
	for {
		ok, err := fileExists(filepath.Join(goModPath, "go.mod"))
		if err != nil {
			return "", err
		}
		if ok {
			break
		}
		goModPath = filepath.Dir(goModPath)
	}
	return goModPath, nil
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
