package pillow

import (
	"database/sql"
	"fmt"
	"github.com/eatmoreapple/pillow/driver"
	"log"
	"os"
)

// Engine is the main struct of pillow
type Engine struct {
	Configuration *Configuration
	Driver        driver.Driver
	DB            *sql.DB
	Logger        *log.Logger
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

func (e *Engine) init() error {
	env, err := e.Configuration.Environments.DefaultEnv()
	if err != nil {
		return err
	}
	drv, err := driver.Get(env.Driver)
	if err != nil {
		return err
	}
	e.Driver = drv

	e.DB, err = env.Connect()
	if err != nil {
		return err
	}
	return nil
}

func (e *Engine) getMapperStatement(v any) (Statement, error) {
	id, err := FuncForPC(v)
	if err != nil {
		return nil, err
	}

	stat, err := e.Configuration.Mappers.GetStatementByID(id)

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
