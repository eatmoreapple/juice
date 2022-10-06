package pillow

import (
	"database/sql"
	"errors"
	"time"
)

// Environment defines a environment.
// It contains a database connection configuration.
type Environment struct {
	// ID is an identifier of the environment.
	ID string

	// DataSource is a string in a driver-specific format.
	DataSource string

	// Driver is a driver for
	Driver string

	// MaxIdleConnNum is a maximum number of idle connections.
	MaxIdleConnNum int

	// MaxOpenConnNum is a maximum number of open connections.
	MaxOpenConnNum int

	// MaxConnLifetime is a maximum lifetime of a connection.
	MaxConnLifetime int

	// MaxIdleConnLifetime is a maximum lifetime of an idle connection.
	MaxIdleConnLifetime int
}

func (e Environment) Connect() (*sql.DB, error) {
	db, err := sql.Open(e.Driver, e.DataSource)
	if err != nil {
		return nil, err
	}
	if e.MaxIdleConnNum > 0 {
		db.SetMaxIdleConns(e.MaxIdleConnNum)
	}
	if e.MaxOpenConnNum > 0 {
		db.SetMaxOpenConns(e.MaxOpenConnNum)
	}
	if e.MaxConnLifetime > 0 {
		db.SetConnMaxLifetime(time.Duration(e.MaxConnLifetime) * time.Second)
	}
	if e.MaxIdleConnLifetime > 0 {
		db.SetConnMaxLifetime(time.Duration(e.MaxIdleConnLifetime) * time.Second)
	}
	return db, nil
}

// Environments is a collection of environments.
type Environments struct {
	// Default is an identifier of the default environment.
	Default string

	// envs is a map of environments.
	// The key is an identifier of the environment.
	envs map[string]*Environment
}

// DefaultEnv returns the default environment.
func (e *Environments) DefaultEnv() (*Environment, error) {
	return e.Use(e.Default)
}

// Use returns the environment specified by the identifier.
func (e *Environments) Use(id string) (*Environment, error) {
	env, exists := e.envs[id]
	if !exists {
		return nil, errors.New("environment not found")
	}
	return env, nil
}
