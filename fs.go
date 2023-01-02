package juice

import (
	"io/fs"
	"os"
	"path/filepath"
)

// LocalFS is a file system.
type LocalFS struct{}

// Open implements fs.FS.
func (f LocalFS) Open(name string) (fs.File, error) {
	return os.Open(name)
}

type fsWrapper struct {
	fs.FS
	baseDir string
}

func (f fsWrapper) Open(name string) (fs.File, error) {
	path := filepath.Join(f.baseDir, name)
	return f.FS.Open(path)
}
