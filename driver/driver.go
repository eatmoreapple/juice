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

package driver

import (
	"fmt"
	"strconv"
	"sync"
)

// Driver is a driver of database.
type Driver interface {
	// Translator returns a translator of SQL.
	Translator() Translator
}

var (
	// registeredDrivers is a map of registered drivers.
	// The key is a name of driver, it is used to get a driver.
	registeredDrivers = make(map[string]Driver)

	// lock is a lock for registeredDrivers.
	// For thread safety.
	lock sync.RWMutex
)

// Register registers a driver.
// The name is used to get a driver.
func Register(name string, driver Driver) {
	if driver == nil {
		panic("driver: Register driver is nil")
	}
	lock.Lock()
	defer lock.Unlock()
	// allow re-registration
	registeredDrivers[name] = driver
}

// Get returns a driver of the name.
// If the name is not registered, it returns an error.
func Get(name string) (Driver, error) {
	lock.RLock()
	defer lock.RUnlock()
	driver, ok := registeredDrivers[name]
	if !ok {
		return nil, fmt.Errorf("driver %s not found", name)
	}
	return driver, nil
}

// MySQLDriver is a driver of MySQL.
type MySQLDriver struct{}

// Translator returns a translator of SQL.
func (d MySQLDriver) Translator() Translator {
	return TranslateFunc(func(matched string) string { return "?" })
}

func (d MySQLDriver) String() string {
	return "mysql"
}

// SQLiteDriver is a driver of SQLite.
type SQLiteDriver struct{}

// Translator returns a translator of SQL.
func (d SQLiteDriver) Translator() Translator {
	return TranslateFunc(func(matched string) string { return "?" })
}

func (d SQLiteDriver) String() string {
	return "sqlite3"
}

// PostgresDriver is a driver of PostgreSQL.
type PostgresDriver struct{}

// Translator is a function to translate a matched string.
func (d PostgresDriver) Translator() Translator {
	var i int
	return TranslateFunc(func(matched string) string {
		i++
		return "$" + strconv.Itoa(i)
	})
}

func (d PostgresDriver) String() string {
	return "postgres"
}

// OracleDriver is a driver of Oracle.
type OracleDriver struct{}

// Translator is a function to translate a matched string.
func (o OracleDriver) Translator() Translator {
	var i int
	return TranslateFunc(func(matched string) string {
		i++
		return ":" + strconv.Itoa(i)
	})
}

func (o OracleDriver) String() string {
	return "oracle"
}

func init() {
	Register("mysql", &MySQLDriver{})
	Register("sqlite3", &SQLiteDriver{})
	Register("postgres", &PostgresDriver{})
	Register("oracle", &OracleDriver{})
}
