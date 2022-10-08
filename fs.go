package juice

import (
	"io/fs"
	"os"
)

// LocalFS is a file system.
type LocalFS struct{}

// Open implements fs.FS.
func (f LocalFS) Open(name string) (fs.File, error) {
	return os.Open(name)
}
