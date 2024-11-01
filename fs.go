/*
Copyright 2023 eatmoreapple

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package juice

import (
	"io/fs"
	"os"
	unixpath "path"
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

// Open opens the named file within the base directory of the fsWrapper.
// It joins the base directory and the name using Unix-style path separators,
// ensuring compatibility with io/fs.Open which uses slash-separated paths on all systems.
func (f fsWrapper) Open(name string) (fs.File, error) {
	path := unixpath.Join(f.baseDir, name)
	return f.fs.Open(path)
}
