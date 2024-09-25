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

// SQLiteDriver is a driver of SQLite.
type SQLiteDriver struct{}

// Translator returns a translator of SQL.
func (d SQLiteDriver) Translator() Translator {
	return TranslateFunc(func(matched string) string { return "?" })
}

func (d SQLiteDriver) String() string {
	return "sqlite3"
}

func init() {
	Register("sqlite3", &SQLiteDriver{})
}
