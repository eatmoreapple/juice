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

// IConfiguration is the interface of configuration.
type IConfiguration interface {
	// Environments returns the environments.
	Environments() *Environments

	// Settings returns the settings.
	Settings() SettingProvider

	// GetStatement returns the xmlSQLStatement of the given value.
	GetStatement(v any) (Statement, error)
}

// Configuration is a configuration of juice.
type Configuration struct {
	// environments is a map of environments.
	environments Environments

	// mappers is a map of mappers.
	mappers *Mappers

	// settings is a map of settings.
	settings keyValueSettingProvider
}

// Environments returns the environments.
func (c Configuration) Environments() *Environments {
	return &c.environments
}

// Settings returns the settings.
func (c Configuration) Settings() SettingProvider {
	return &c.settings
}

// GetStatement returns the xmlSQLStatement of the given value.
func (c Configuration) GetStatement(v any) (Statement, error) {
	return c.mappers.GetStatement(v)
}
