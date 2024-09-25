/*
Copyright 2024 eatmoreapple

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

package driver

// MySQLDriver is a driver of MySQL.
type MySQLDriver struct{}

// Translator returns a translator of SQL.
func (d MySQLDriver) Translator() Translator {
	return TranslateFunc(func(matched string) string { return "?" })
}

func (d MySQLDriver) String() string {
	return "mysql"
}

func init() {
	Register("mysql", &MySQLDriver{})
}
