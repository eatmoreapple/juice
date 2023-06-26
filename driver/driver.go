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
type SimpleDriver struct{}

// Translate returns a translator of SQL.
func (d SimpleDriver) Translator() Translator {
	return TranslateFunc(func(matched string) string {
		return "?"
	})
}

// MySQLDriver is a driver of MySQL.
type MySQLDriver struct {
	SimpleDriver
}

func (d MySQLDriver) String() string {
	return "mysql"
}

// SQLiteDriver is a driver of SQLite.
type SQLiteDriver struct {
	SimpleDriver
}

func (d SQLiteDriver) String() string {
	return "sqlite"
}

// PostgresDriver is a driver of PostgreSQL.
type PostgresDriver struct{}

// Translate is a function to translate a matched string.
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

// Translate is a function to translate a matched string.
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
	Register("sqlite", &SQLiteDriver{})
	Register("postgres", &PostgresDriver{})
	Register("oracle", &OracleDriver{})
}
