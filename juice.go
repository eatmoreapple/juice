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
	"sync"
)

// Engine is the implementation of Manager interface and the core of juice.
type Engine struct {
	// configuration is the configuration of the engine
	// It is used to initialize the engine and to one the mapper statements
	configuration *Configuration

	// driver is the driver used by the engine
	// It is used to initialize the database connection and translate the mapper statements
	driver driver.Driver

	// db is the database connection
	db *sql.DB

	// rw is the read write lock
	// default use sync.RWMutex
	// if you have many multiple processes to access the same database,
	// you can use the distributed lock, such as redis, etc.
	// call SetLocker to set your own business lock.
	rw RWLocker

	// middlewares is the middlewares of the engine
	// It is used to intercept the execution of the statements
	// like logging, tracing, etc.
	middlewares MiddlewareGroup

	// executorWrapper is the wrapper of the executor
	// which is used to wrap the executor
	executorWrapper ExecutorWrapper

	// cacheFactory is the cache factory of the engine
	cacheFactory func() cache.Cache
}

// executor represents a mapper executor with the given parameters
func (e *Engine) executor(v any) (*executor, error) {
	stat, err := e.GetConfiguration().Mappers.GetStatement(v)
	if err != nil {
		return nil, err
	}
	return &executor{statement: stat, session: e.DB()}, nil
}

// Object implements the Manager interface
func (e *Engine) Object(v any) Executor {
	exe, err := e.executor(v)
	if err != nil {
		return inValidExecutor(err)
	}
	return e.executorWrapper.WarpExecutor(exe)
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

// CacheTx returns a TxCacheManager.
func (e *Engine) CacheTx() TxCacheManager {
	return e.ContextCacheTx(context.Background(), nil)
}

// ContextCacheTx returns a TxCacheManager with the given context.
func (e *Engine) ContextCacheTx(ctx context.Context, opt *sql.TxOptions) TxCacheManager {
	tx := e.ContextTx(ctx, opt)
	return NewTxCacheManager(tx, e.cacheFactory())
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
	cfg.engine = e
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

// SetCacheFactory sets the cache factory of the engine.
func (e *Engine) SetCacheFactory(factory func() cache.Cache) {
	if factory == nil {
		panic("cache factory is nil")
	}
	e.cacheFactory = factory
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
	env, err := e.configuration.Environments.DefaultEnv()
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
	e.db, err = env.Connect(drv)
	return err
}

// NewEngine creates a new Engine
func NewEngine(configuration *Configuration) (*Engine, error) {
	engine := &Engine{}
	engine.SetLocker(&sync.RWMutex{})
	engine.SetConfiguration(configuration)
	if err := engine.init(); err != nil {
		return nil, err
	}
	// set default cache factory
	engine.SetCacheFactory(func() cache.Cache { return cache.New() })
	// add the default middlewares
	engine.Use(&useGeneratedKeysMiddleware{})
	// set default executor wrapper
	engine.executorWrapper = ExecutorWarpGroup{
		NewSessionCtxInjectorExecutorWrapper(),
		NewParamCtxInjectorExecutorWarpper(),
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
