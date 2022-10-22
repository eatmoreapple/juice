package juice

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/eatmoreapple/juice/driver"
)

// Engine is the main struct of pillow
type Engine struct {

	// configuration is the configuration of the engine
	// It is used to initialize the engine and to get the mapper statements
	configuration *Configuration

	// Driver is the driver used by the engine
	// It is used to initialize the database connection and translate the mapper statements
	Driver driver.Driver

	// DB is the database connection
	DB *sql.DB

	// rw is the read write lock
	rw sync.RWMutex

	// middlewares is the middlewares of the engine
	// It is used to intercept the execution of the statements
	// like logging, tracing, etc.
	middlewares MiddlewareGroup
}

// Object implements the Manager interface
func (e *Engine) Object(v interface{}) Executor {
	stat, err := e.getMapperStatement(v)
	if err != nil {
		return inValidExecutor(err)
	}
	return &executor{engine: e, statement: stat, session: e.DB}
}

// Tx returns a TxManager
func (e *Engine) Tx() TxManager {
	tx, err := e.DB.Begin()
	return &txManager{manager: e, tx: tx, err: err}
}

// ContextTx returns a TxManager with the given context
func (e *Engine) ContextTx(ctx context.Context, opt *sql.TxOptions) TxManager {
	tx, err := e.DB.BeginTx(ctx, opt)
	return &txManager{manager: e, tx: tx, err: err}
}

// GetConfiguration returns the configuration of the engine
func (e *Engine) GetConfiguration() *Configuration {
	e.rw.RLock()
	defer e.rw.RUnlock()
	return e.configuration
}

// SetConfiguration sets the configuration of the engine
func (e *Engine) SetConfiguration(cfg *Configuration) {
	e.rw.Lock()
	defer e.rw.Unlock()
	e.configuration = cfg
}

// Use adds a middleware to the engine
func (e *Engine) Use(middleware Middleware) {
	e.middlewares = append(e.middlewares, middleware)
}

// init initializes the engine
func (e *Engine) init() error {

	// get the default environment from the configuration
	env, err := e.configuration.Environments.DefaultEnv()
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
func (e *Engine) getMapperStatement(v any) (stat *Statement, err error) {
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
	stat, err = e.GetConfiguration().Mappers.GetStatementByID(id)

	if err != nil {
		return nil, fmt.Errorf("mapper %s not found", id)
	}
	return stat, nil
}

// NewEngine creates a new Engine
func NewEngine(configuration *Configuration) (*Engine, error) {
	engine := &Engine{}
	engine.SetConfiguration(configuration)
	if err := engine.init(); err != nil {
		return nil, err
	}
	return engine, nil
}

// DefaultEngine is the default engine
// It adds an interceptor to log the statements
func DefaultEngine(configuration *Configuration) (*Engine, error) {
	engine, err := NewEngine(configuration)
	if err != nil {
		return nil, err
	}
	engine.Use(&DebugMiddleware{})
	return engine, nil
}
