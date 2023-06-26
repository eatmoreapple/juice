package juice

import (
	"context"
	"database/sql"
	"errors"
	"github.com/eatmoreapple/juice/cache"
	"log"
	"reflect"
	"strconv"
	"time"
)

// Middleware is a wrapper of QueryHandler and ExecHandler.
type Middleware interface {
	// QueryContext wraps the QueryHandler.
	QueryContext(stmt *Statement, next QueryHandler) QueryHandler
	// ExecContext wraps the ExecHandler.
	ExecContext(stmt *Statement, next ExecHandler) ExecHandler
}

// ensure MiddlewareGroup implements Middleware.
var _ Middleware = MiddlewareGroup(nil) // compile time check

// MiddlewareGroup is a group of Middleware.
type MiddlewareGroup []Middleware

// QueryContext implements Middleware.
// Call QueryContext will call all the QueryContext of the middlewares in the group.
func (m MiddlewareGroup) QueryContext(stmt *Statement, next QueryHandler) QueryHandler {
	for _, middleware := range m {
		next = middleware.QueryContext(stmt, next)
	}
	return next
}

// ExecContext implements Middleware.
// Call ExecContext will call all the ExecContext of the middlewares in the group.
func (m MiddlewareGroup) ExecContext(stmt *Statement, next ExecHandler) ExecHandler {
	for _, middleware := range m {
		next = middleware.ExecContext(stmt, next)
	}
	return next
}

// logger is a default logger for debug.
var logger = log.New(log.Writer(), "[juice] ", log.Flags())

// ensure DebugMiddleware implements Middleware.
var _ Middleware = (*DebugMiddleware)(nil) // compile time check

// DebugMiddleware is a middleware that prints the sql statement and the execution time.
type DebugMiddleware struct{}

// QueryContext implements Middleware.
// QueryContext will print the sql statement and the execution time.
func (m *DebugMiddleware) QueryContext(stmt *Statement, next QueryHandler) QueryHandler {
	if !m.isBugMode(stmt) {
		return next
	}
	// wrapper QueryHandler
	return func(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
		start := time.Now()
		rows, err := next(ctx, query, args...)
		spent := time.Since(start)
		logger.Printf("\x1b[33m[%s]\x1b[0m \x1b[32m %s\x1b[0m \x1b[34m %v\x1b[0m \x1b[31m %v\x1b[0m\n", stmt.Name(), query, args, spent)
		return rows, err
	}
}

// ExecContext implements Middleware.
// ExecContext will print the sql statement and the execution time.
func (m *DebugMiddleware) ExecContext(stmt *Statement, next ExecHandler) ExecHandler {
	if !m.isBugMode(stmt) {
		return next
	}
	// wrapper ExecContext
	return func(ctx context.Context, query string, args ...any) (sql.Result, error) {
		start := time.Now()
		rows, err := next(ctx, query, args...)
		spent := time.Since(start)
		logger.Printf("\x1b[33m[%s]\x1b[0m \x1b[32m %s\x1b[0m \x1b[34m %v\x1b[0m \x1b[31m %v\x1b[0m\n", stmt.Name(), query, args, spent)
		return rows, err
	}
}

// isBugMode returns true if the debug mode is on.
// Default debug mode is on.
// You can turn off the debug mode by setting the debug tag to false in the mapper statement attribute or the configuration.
func (m *DebugMiddleware) isBugMode(stmt *Statement) bool {
	// try to one the bug mode from the Statement
	debug := stmt.Attribute("debug")
	// if the bug mode is not set, try to one the bug mode from the Context
	if debug == "false" {
		return false
	}
	if cfg := stmt.Configuration(); cfg.Settings.Get("debug") == "false" {
		return false
	}
	return true
}

// ensure TimeoutMiddleware implements Middleware
var _ Middleware = (*TimeoutMiddleware)(nil) // compile time check

// TimeoutMiddleware is a middleware that sets the timeout for the sql statement.
type TimeoutMiddleware struct{}

// QueryContext implements Middleware.
// QueryContext will set the timeout for the sql statement.
func (t TimeoutMiddleware) QueryContext(stmt *Statement, next QueryHandler) QueryHandler {
	timeout := t.getTimeout(stmt)
	if timeout <= 0 {
		return next
	}
	return func(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
		ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
		defer cancel()
		return next(ctx, query, args...)
	}
}

// ExecContext implements Middleware.
// ExecContext will set the timeout for the sql statement.
func (t TimeoutMiddleware) ExecContext(stmt *Statement, next ExecHandler) ExecHandler {
	timeout := t.getTimeout(stmt)
	if timeout <= 0 {
		return next
	}
	return func(ctx context.Context, query string, args ...any) (sql.Result, error) {
		ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
		defer cancel()
		return next(ctx, query, args...)
	}
}

// getTimeout returns the timeout from the Statement.
func (t TimeoutMiddleware) getTimeout(stmt *Statement) (timeout int64) {
	timeoutAttr := stmt.Attribute("timeout")
	if timeoutAttr == "" {
		return
	}
	timeout, _ = strconv.ParseInt(timeoutAttr, 10, 64)
	return
}

// GenericMiddleware defines the middleware interface for the generic execution.
type GenericMiddleware[T any] interface {
	// QueryContext wraps the GenericQueryHandler.
	// The GenericQueryHandler is a function that accepts a context.Context, a query string and a slice of arguments.
	QueryContext(stmt *Statement, next GenericQueryHandler[T]) GenericQueryHandler[T]

	// ExecContext wraps the ExecHandler.
	// The ExecHandler is a function that accepts a context.Context, a query string and a slice of arguments.
	ExecContext(stmt *Statement, next ExecHandler) ExecHandler
}

// ensure GenericMiddlewareGroup implements GenericMiddleware
var _ GenericMiddleware[any] = (GenericMiddlewareGroup[any])(nil) // compile time check

// GenericMiddlewareGroup is a group of GenericMiddleware.
// It implements the GenericMiddleware interface.
type GenericMiddlewareGroup[T any] []GenericMiddleware[T]

// QueryContext implements GenericMiddleware.
func (m GenericMiddlewareGroup[T]) QueryContext(stmt *Statement, next GenericQueryHandler[T]) GenericQueryHandler[T] {
	for _, middleware := range m {
		next = middleware.QueryContext(stmt, next)
	}
	return next
}

// ExecContext implements GenericMiddleware.
func (m GenericMiddlewareGroup[T]) ExecContext(stmt *Statement, next ExecHandler) ExecHandler {
	for _, middleware := range m {
		next = middleware.ExecContext(stmt, next)
	}
	return next
}

// ensure GenericMiddlewareGroup implements GenericMiddleware
var _ GenericMiddleware[any] = (*CacheMiddleware[any])(nil) // compile time check

// CacheMiddleware is a middleware that caches the result of the sql query.
type CacheMiddleware[T any] struct {
	cache cache.Cache
}

// QueryContext implements Middleware.
func (c *CacheMiddleware[T]) QueryContext(stmt *Statement, next GenericQueryHandler[T]) GenericQueryHandler[T] {
	if c.cache == nil {
		return next
	}
	return func(ctx context.Context, query string, args ...any) (result T, err error) {
		// ptr is the pointer of the result, it is the destination of the binding.
		var ptr any = &result

		rv := reflect.ValueOf(result)

		// if the result is a pointer, create a new instance of the element.
		// you'd better not use a nil pointer as the result.
		if rv.Kind() == reflect.Ptr {
			result = reflect.New(rv.Type().Elem()).Interface().(T)
			ptr = result
		}

		// If the cache is enabled and cache is not disabled in this statement.
		if stmt.Attribute("useCache") != "false" {
			// cacheKey is the key which is used to get the result and put the result to the cache.
			var cacheKey string

			// CacheKeyFunc is the function which is used to generate the cache key.
			// default is the md5 of the query and args.
			// reset the CacheKeyFunc variable to change the default behavior.
			cacheKey, err = CacheKeyFunc(stmt, query, args)
			if err != nil {
				return
			}

			// try to get the result from the cache
			// if the result is found, return it directly.
			if err = c.cache.Get(ctx, cacheKey, ptr); err == nil {
				return
			}

			// ErrCacheNotFound means the cache is not found,
			// we should continue to query the database.
			if !errors.Is(err, cache.ErrCacheNotFound) {
				return
			}
			// put the result to the cache
			defer func() {
				if err == nil {
					err = c.cache.Set(ctx, cacheKey, result)
				}
			}()
		}

		return next(ctx, query, args...)
	}
}

func (c *CacheMiddleware[T]) ExecContext(stmt *Statement, next ExecHandler) ExecHandler {
	// if the cache is enabled and flushCache is not disabled in this statement.
	flushCache := stmt.Attribute("flushCache") != "false" && c.cache != nil
	if !flushCache {
		return next
	}
	return func(ctx context.Context, query string, args ...any) (sql.Result, error) {
		// call the next handler
		result, err := next(ctx, query, args...)
		if err == nil {
			err = c.cache.Flush(ctx)
		}
		return result, err
	}
}
