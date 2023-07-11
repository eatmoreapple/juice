package juice

import (
	"database/sql"
	"fmt"
	"github.com/eatmoreapple/juice/driver"
	"os"
	"time"
)

// Environment defines a environment.
// It contains a database connection configuration.
type Environment struct {
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

	// attrs is a map of attributes.
	attrs map[string]string
}

// setAttr sets a value of the attribute.
func (e *Environment) setAttr(key, value string) {
	if e.attrs == nil {
		e.attrs = make(map[string]string)
	}
	e.attrs[key] = value
}

// Attr returns a value of the attribute.
func (e *Environment) Attr(key string) string {
	return e.attrs[key]
}

// ID returns a identifier of the environment.
func (e *Environment) ID() string {
	return e.Attr("id")
}

// provider is a environment value provider.
// It provides a value of the environment variable.
func (e *Environment) provider() EnvValueProvider {
	return GetEnvValueProvider(e.Attr("provider"))
}

// Connect returns a database connection.
func (e *Environment) Connect(driver driver.Driver) (*sql.DB, error) {
	// Open a database connection with the given driver.
	db, err := driver.Open(e.DataSource)
	if err != nil {
		return nil, err
	}

	// Set connection parameters.

	// set max idle connection number if it is specified.
	if e.MaxIdleConnNum > 0 {
		db.SetMaxIdleConns(e.MaxIdleConnNum)
	}

	// set max open connection number if it is specified.
	if e.MaxOpenConnNum > 0 {
		db.SetMaxOpenConns(e.MaxOpenConnNum)
	}

	// set max connection lifetime if it is specified.
	if e.MaxConnLifetime > 0 {
		db.SetConnMaxLifetime(time.Duration(e.MaxConnLifetime) * time.Second)
	}

	// set max idle connection lifetime if it is specified.
	if e.MaxIdleConnLifetime > 0 {
		db.SetConnMaxLifetime(time.Duration(e.MaxIdleConnLifetime) * time.Second)
	}
	return db, nil
}

// Environments is a collection of environments.
type Environments struct {
	attr map[string]string

	// envs is a map of environments.
	// The key is an identifier of the environment.
	envs map[string]*Environment
}

// setAttr sets a value of the attribute.
func (e *Environments) setAttr(key, value string) {
	if e.attr == nil {
		e.attr = make(map[string]string)
	}
	e.attr[key] = value
}

// Attr returns a value of the attribute.
func (e *Environments) Attr(key string) string {
	return e.attr[key]
}

// DefaultEnv returns the default environment.
func (e *Environments) DefaultEnv() (*Environment, error) {
	return e.Use(e.Attr("default"))
}

// Use returns the environment specified by the identifier.
func (e *Environments) Use(id string) (*Environment, error) {
	env, exists := e.envs[id]
	if !exists {
		return nil, fmt.Errorf("environment %s not found", id)
	}
	return env, nil
}

// EnvValueProvider defines a environment value provider.
type EnvValueProvider interface {
	Get(key string) (string, error)
}

// envValueProviderLibraries is a map of environment value providers.
var envValueProviderLibraries = map[string]EnvValueProvider{}

// EnvValueProviderFunc is a function type of environment value provider.
type EnvValueProviderFunc func(key string) (string, error)

// Get is a function type of environment value provider.
func (f EnvValueProviderFunc) Get(key string) (string, error) {
	return f(key)
}

// OsEnvValueProvider is a environment value provider that uses os.Getenv.
type OsEnvValueProvider struct{}

// Get returns a value of the environment variable.
// It uses os.Getenv.
func (p OsEnvValueProvider) Get(key string) (string, error) {
	var err error
	key = formatRegexp.ReplaceAllStringFunc(key, func(find string) string {
		value := os.Getenv(formatRegexp.FindStringSubmatch(find)[1])
		if len(value) == 0 {
			err = fmt.Errorf("environment variable %s not found", find)
		}
		return value
	})
	return key, err
}

// RegisterEnvValueProvider registers an environment value provider.
// The key is a name of the provider.
// The value is a provider.
// It allows to override the default provider.
func RegisterEnvValueProvider(name string, provider EnvValueProvider) {
	if len(name) == 0 {
		panic("name is empty")
	}
	if provider == nil {
		panic("juice: environment value provider is nil")
	}
	envValueProviderLibraries[name] = provider
}

// defaultEnvValueProvider is a default environment value provider.
var defaultEnvValueProvider EnvValueProviderFunc = func(key string) (string, error) { return key, nil }

// GetEnvValueProvider returns a environment value provider.
func GetEnvValueProvider(key string) EnvValueProvider {
	if provider, exists := envValueProviderLibraries[key]; exists {
		return provider
	}
	return defaultEnvValueProvider
}

func init() {
	// Register the default environment value provider.
	RegisterEnvValueProvider("env", &OsEnvValueProvider{})
}
