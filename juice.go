package juice

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/eatmoreapple/juice/driver"
)

// Engine is the main struct of pillow
type Engine struct {

	// Configuration is the configuration of the engine
	// It is used to initialize the engine and to get the mapper statements
	Configuration *Configuration

	// Driver is the driver used by the engine
	// It is used to initialize the database connection and translate the mapper statements
	Driver driver.Driver

	// DB is the database connection
	DB *sql.DB

	// Logger is the logger used by the engine
	Logger *log.Logger
}

// Statement implements the Statement interface
func (e *Engine) Statement(v interface{}) Executor {
	stat, err := e.getMapperStatement(v)
	return &executor{err: err, engine: e, statement: stat, session: e.DB}
}

// Tx returns a TxMapperExecutor
func (e *Engine) Tx() TxMapperExecutor {
	tx, err := e.DB.Begin()
	return &txStatement{engine: e, tx: tx, err: err}
}

// init initializes the engine
func (e *Engine) init() error {

	// get the default environment from the configuration
	env, err := e.Configuration.Environments.DefaultEnv()
	if err != nil {
		return err
	}

	// try to get the driver from the configuration
	drv, err := driver.Get(env.Driver)
	if err != nil {
		return err
	}
	e.Driver = drv

	// open the database connection
	e.DB, err = env.Connect()
	if err != nil {
		return err
	}
	return nil
}

// try to get the statement from the configuration with the given interface
func (e *Engine) getMapperStatement(v any) (stat Statement, err error) {
	var id string

	// if the interface is a string, use it as the id
	if str, ok := v.(string); ok {
		id = str
	} else {
		// else try to get the id from the interface
		if id, err = FuncForPC(v); err != nil {
			return nil, err
		}
	}

	// try to get the statement from the configuration
	stat, err = e.Configuration.Mappers.GetStatementByID(id)

	if err != nil {
		return nil, fmt.Errorf("mapper %s not found", id)
	}
	return stat, nil
}

// DefaultEngine is the default engine
// It is initialized with the default configuration
func DefaultEngine(configuration *Configuration) (*Engine, error) {
	engine, err := NewEngine(configuration)
	if err != nil {
		return nil, err
	}
	engine.Logger = log.New(os.Stdout, "[Pillow] ", log.LstdFlags)
	return engine, nil
}

// NewEngine creates a new Engine
func NewEngine(configuration *Configuration) (*Engine, error) {
	engine := &Engine{
		Configuration: configuration,
	}
	if err := engine.init(); err != nil {
		return nil, err
	}
	return engine, nil
}
