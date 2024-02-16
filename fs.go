package juice

import (
	"io/fs"
	"os"
	stdpath "path"
	"path/filepath"
)

// localFS is a file system.
type localFS struct {
	baseDir string
}

// Open implements fs.FS.
func (f localFS) Open(name string) (fs.File, error) {
	path := filepath.Join(f.baseDir, name)
	return os.Open(path)
}

type fsWrapper struct {
	fs      fs.FS
	baseDir string
}

func (f fsWrapper) Open(name string) (fs.File, error) {
	path := stdpath.Join(f.baseDir, name)
	return f.fs.Open(path)
}
