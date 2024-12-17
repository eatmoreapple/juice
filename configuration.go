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
	"path"
	"path/filepath"
)

// IConfiguration is the interface of configuration.
type IConfiguration interface {
	// Environments returns the environments.
	Environments() EnvironmentProvider

	// Settings returns the settings.
	Settings() SettingProvider

	// GetStatement returns the xmlSQLStatement of the given value.
	GetStatement(v any) (Statement, error)
}

// Configuration is a configuration of juice.
type Configuration struct {
	// environments is a map of environments.
	environments *environments

	// mappers is a map of mappers.
	mappers *Mappers

	// settings is a map of settings.
	settings keyValueSettingProvider
}

// Environments returns the environments.
func (c Configuration) Environments() EnvironmentProvider {
	return c.environments
}

// Settings returns the settings.
func (c Configuration) Settings() SettingProvider {
	return &c.settings
}

// GetStatement returns the xmlSQLStatement of the given value.
func (c Configuration) GetStatement(v any) (Statement, error) {
	return c.mappers.GetStatement(v)
}

func NewXMLConfiguration(filename string) (IConfiguration, error) {
	return newLocalXMLConfiguration(filename, false)
}

// for go linkname
func newLocalXMLConfiguration(filename string, ignoreEnv bool) (IConfiguration, error) {
	baseDir := filepath.Dir(filename)
	filename = filepath.Base(filename)
	return newXMLConfigurationParser(localFS{baseDir: baseDir}, filename, ignoreEnv)
}

// NewXMLConfigurationWithFS creates a new Configuration from an XML file.
func NewXMLConfigurationWithFS(fs fs.FS, filename string) (IConfiguration, error) {
	baseDir := path.Dir(filename)
	filename = path.Base(filename)
	return newXMLConfigurationParser(fsWrapper{baseDir: baseDir, fs: fs}, filename, false)
}

// newXMLConfigurationParser creates a new Configuration from an XML file which ignores environment parsing.
// for internal use only.
func newXMLConfigurationParser(fs fs.FS, filename string, ignoreEnv bool) (IConfiguration, error) {
	file, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	parser := &XMLParser{FS: fs, ignoreEnv: ignoreEnv}
	parser.AddXMLElementParser(
		&XMLEnvironmentsElementParser{},
		&XMLMappersElementParser{},
		&XMLSettingsElementParser{},
	)
	return parser.Parse(file)
}
