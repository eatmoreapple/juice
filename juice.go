package juice

import (
	"context"
	"database/sql"
	"sync"

	"github.com/eatmoreapple/juice/driver"
)

// Engine is the main struct of pillow
type Engine struct {

	// configuration is the configuration of the engine
	// It is used to initialize the engine and to one the mapper statements
	configuration *Configuration

	// Driver is the driver used by the engine
	// It is used to initialize the database connection and translate the mapper statements
	Driver driver.Driver

	// db is the database connection
	db *sql.DB

	// rw is the read write lock
	rw sync.RWMutex

	// middlewares is the middlewares of the engine
	// It is used to intercept the execution of the statements
	// like logging, tracing, etc.
	middlewares MiddlewareGroup
}

// Object implements the Manager interface
func (e *Engine) Object(v any) Executor {
	exe, err := e.executor(v)
	if err != nil {
		return inValidExecutor(err)
	}
	exe.session = e.DB()
	return exe
}

func (e *Engine) executor(v any) (*executor, error) {
	stat, err := e.GetConfiguration().Mappers.GetStatement(v)
	if err != nil {
		return nil, err
	}
	stat.engine = e
	return &executor{engine: e, statement: stat}, nil
}

// Tx returns a TxManager
func (e *Engine) Tx() TxManager {
	return e.ContextTx(context.Background(), nil)
}

// ContextTx returns a TxManager with the given context
func (e *Engine) ContextTx(ctx context.Context, opt *sql.TxOptions) TxManager {
	tx, err := e.DB().BeginTx(ctx, opt)
	return &txManager{engine: e, tx: tx, err: err}
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

func (e *Engine) DB() *sql.DB {
	return e.db
}

// init initializes the engine
func (e *Engine) init() error {

	// one the default environment from the configuration
	env, err := e.configuration.Environments.DefaultEnv()
	if err != nil {
		return err
	}

	// try to one the driver from the configuration
	drv, err := driver.Get(env.Driver)
	if err != nil {
		return err
	}
	e.Driver = drv

	// open the database connection
	e.db, err = env.Connect()
	return err
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
	engine.Use(&TimeoutMiddleware{})
	engine.Use(&DebugMiddleware{})
	return engine, nil
}
