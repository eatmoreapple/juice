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

package juice

import (
	"context"
	"database/sql"
	"github.com/eatmoreapple/juice/cache"
	"github.com/eatmoreapple/juice/driver"
)

// Engine is the implementation of Manager interface and the core of juice.
type Engine struct {
	// configuration is the configuration of the engine
	// It is used to initialize the engine and to one the mapper statements
	configuration IConfiguration

	// driver is the driver used by the engine
	// It is used to initialize the database connection and translate the mapper statements
	driver driver.Driver

	// db is the database connection
	db *sql.DB

	// rw is the read write lock
	// default use the no-op locker
	// if you have many multiple processes to access the same database,
	// you can use the distributed lock, such as redis, etc.
	// call SetLocker to set your own business lock.
	rw RWLocker

	// middlewares is the middlewares of the engine
	// It is used to intercept the execution of the statements
	// like logging, tracing, etc.
	middlewares MiddlewareGroup
}

// executor represents a mapper executor with the given parameters
func (e *Engine) executor(v any) (*executor, error) {
	stat, err := e.GetConfiguration().GetStatement(v)
	if err != nil {
		return nil, err
	}
	return &executor{
		statement:   stat,
		session:     e.DB(),
		driver:      e.driver,
		middlewares: e.middlewares,
	}, nil
}

// Object implements the Manager interface
func (e *Engine) Object(v any) Executor {
	exe, err := e.executor(v)
	if err != nil {
		return inValidExecutor(err)
	}
	return exe
}

// Tx returns a TxManager
func (e *Engine) Tx() TxManager {
	return e.ContextTx(context.Background(), nil)
}

// ContextTx returns a TxManager with the given context
func (e *Engine) ContextTx(ctx context.Context, opt *sql.TxOptions) TxManager {
	tx, err := e.DB().BeginTx(ctx, opt)
	if err != nil {
		return invalidTxManager{err: err}
	}
	return &txManager{engine: e, tx: tx}
}

// CacheTx returns a TxCacheManager.
func (e *Engine) CacheTx() TxCacheManager {
	return e.ContextCacheTx(context.Background(), nil)
}

// ContextCacheTx returns a TxCacheManager with the given context.
func (e *Engine) ContextCacheTx(ctx context.Context, opt *sql.TxOptions) TxCacheManager {
	tx := e.ContextTx(ctx, opt)
	return NewTxCacheManager(tx, cache.InMemoryScopeCache())
}

// GetConfiguration returns the configuration of the engine
func (e *Engine) GetConfiguration() IConfiguration {
	e.rw.RLock()
	defer e.rw.RUnlock()
	return e.configuration
}

// SetConfiguration sets the configuration of the engine
func (e *Engine) SetConfiguration(cfg IConfiguration) {
	e.rw.Lock()
	defer e.rw.Unlock()
	e.configuration = cfg
}

// Use adds a middleware to the engine
func (e *Engine) Use(middleware Middleware) {
	e.middlewares = append(e.middlewares, middleware)
}

// DB returns the database connection of the engine
func (e *Engine) DB() *sql.DB {
	return e.db
}

// Driver returns the driver of the engine
func (e *Engine) Driver() driver.Driver {
	return e.driver
}

// Close closes the database connection if it is not nil.
func (e *Engine) Close() error {
	if e.db != nil {
		return e.db.Close()
	}
	return nil
}

// SetLocker sets the locker of the engine
// it is not goroutine safe, so it should be called before the engine is used
func (e *Engine) SetLocker(locker RWLocker) {
	if locker == nil {
		panic("locker is nil")
	}
	e.rw = locker
}

// init initializes the engine
func (e *Engine) init() error {
	// one the default environment from the configuration
	envs := e.configuration.Environments()
	defaultEnvName := envs.Attribute("default")
	env, err := envs.Use(defaultEnvName)
	if err != nil {
		return err
	}
	// try to one the driver from the configuration
	drv, err := driver.Get(env.Driver)
	if err != nil {
		return err
	}
	e.driver = drv
	// open the database connection
	e.db, err = ConnectFromEnv(env)
	return err
}

// NewEngine creates a new Engine
// Deprecated: use New instead
func NewEngine(configuration IConfiguration) (*Engine, error) {
	return New(configuration)
}

// New is the alias of NewEngine
func New(configuration IConfiguration) (*Engine, error) {
	engine := &Engine{}
	// for performance, use the no-op locker by default
	engine.SetLocker(&NoOpRWMutex{})
	engine.SetConfiguration(configuration)
	if err := engine.init(); err != nil {
		return nil, err
	}
	// add the default middlewares
	engine.Use(&useGeneratedKeysMiddleware{})
	return engine, nil
}

// DefaultEngine is the alias of Default
// Deprecated: use Default instead
func DefaultEngine(configuration IConfiguration) (*Engine, error) {
	return Default(configuration)
}

// Default creates a new Engine with the default middlewares
// It adds an interceptor to log the statements
func Default(configuration IConfiguration) (*Engine, error) {
	engine, err := New(configuration)
	if err != nil {
		return nil, err
	}
	engine.Use(&TimeoutMiddleware{})
	engine.Use(&DebugMiddleware{})
	return engine, nil
}
