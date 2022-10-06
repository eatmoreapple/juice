package pillow

import (
	"io/fs"
	"os"
)

// FS is a file system.
type FS struct{}

// Open implements fs.FS.
func (f FS) Open(name string) (fs.File, error) {
	return os.Open(name)
}
