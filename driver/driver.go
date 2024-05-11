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

// SimpleDriver is a driver of MySQL、SQLite.
type SimpleDriver struct {
	Name string
}

// Translator returns a translator of SQL.
func (d SimpleDriver) Translator() Translator {
	return TranslateFunc(func(matched string) string { return "?" })
}

// String returns a name of driver.
func (d SimpleDriver) String() string {
	return d.Name
}

// MySQLDriver is a driver of MySQL.
type MySQLDriver struct {
	SimpleDriver
}

// SQLiteDriver is a driver of SQLite.
type SQLiteDriver struct {
	SimpleDriver
}

// PostgresDriver is a driver of PostgreSQL.
type PostgresDriver struct {
	SimpleDriver
}

// Translator is a function to translate a matched string.
// Rewrite this function to change the translation.
func (d PostgresDriver) Translator() Translator {
	var i int
	return TranslateFunc(func(matched string) string {
		i++
		return "$" + strconv.Itoa(i)
	})
}

// OracleDriver is a driver of Oracle.
type OracleDriver struct {
	SimpleDriver
}

// Translator is a function to translate a matched string.
// Rewrite this function to change the translation.
func (o OracleDriver) Translator() Translator {
	var i int
	return TranslateFunc(func(matched string) string {
		i++
		return ":" + strconv.Itoa(i)
	})
}

func init() {
	Register("mysql", &MySQLDriver{SimpleDriver: SimpleDriver{"mysql"}})
	Register("sqlite3", &SQLiteDriver{SimpleDriver: SimpleDriver{"sqlite3"}})
	Register("postgres", &PostgresDriver{SimpleDriver: SimpleDriver{"postgres"}})
	Register("oracle", &OracleDriver{SimpleDriver: SimpleDriver{"oracle"}})
}
