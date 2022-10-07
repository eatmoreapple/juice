package driver

import (
	"fmt"
	"sync"
)

// Driver is a driver of database.
type Driver interface {
	// Translator returns a translator of SQL.
	Translate() Translator
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
// If the name is already registered, it returns an error.
func Register(name string, driver Driver) error {
	lock.Lock()
	defer lock.Unlock()
	if _, ok := registeredDrivers[name]; ok {
		return fmt.Errorf("driver %s already registered", name)
	}
	registeredDrivers[name] = driver
	return nil
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
func (d MySQLDriver) Translate() Translator {
	return TranslateFunc(func(matched string) string {
		return "?"
	})
}

func (d MySQLDriver) String() string {
	return "mysql"
}

func init() {
	_ = Register("mysql", &MySQLDriver{})
}
