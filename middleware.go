package juice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

// ensure useGeneratedKeysMiddleware implements Middleware
var _ Middleware = (*useGeneratedKeysMiddleware)(nil) // compile time check

// useGeneratedKeysMiddleware is a middleware that set the last insert id to the struct.
type useGeneratedKeysMiddleware struct{}

// QueryContext implements Middleware.
// return the result directly and do nothing.
func (m *useGeneratedKeysMiddleware) QueryContext(_ *Statement, next QueryHandler) QueryHandler {
	return next
}

// ExecContext implements Middleware.
// ExecContext will set the last insert id to the struct.
func (m *useGeneratedKeysMiddleware) ExecContext(stmt *Statement, next ExecHandler) ExecHandler {
	if !stmt.IsInsert() {
		return next
	}
	// If the useGeneratedKeys is not set or false, return the result directly.
	useGeneratedKeys := stmt.Attribute("useGeneratedKeys") == "true" ||
		// If the useGeneratedKeys is not set, but the global useGeneratedKeys is set and true.
		stmt.Configuration().Settings.Get("useGeneratedKeys") == "true"

	if !useGeneratedKeys {
		return next
	}
	return func(ctx context.Context, query string, args ...any) (sql.Result, error) {
		result, err := next(ctx, query, args...)

		if err != nil {
			return nil, err
		}

		// try to get param from context
		// FIXME: the param should be set to the context before the query is executed.
		param := ParamFromContext(ctx)

		if param == nil {
			return nil, errors.New("useGeneratedKeys is true, but the param is nil")
		}

		// checkout the input param
		rv := reflect.ValueOf(param)

		// If the useGeneratedKeys is set and true but the param is not a pointer.
		if rv.Kind() != reflect.Ptr {
			return nil, errors.New("useGeneratedKeys is true, but the param is not a pointer")
		}

		rv = reflect.Indirect(rv)

		// If the useGeneratedKeys is set and true but the param is not a struct pointer.
		if rv.Kind() != reflect.Struct {
			return nil, errors.New("useGeneratedKeys is true, but the param is not a struct pointer")
		}

		var field reflect.Value

		// keyProperty is the name of the field that will be set the generated key.
		keyProperty := stmt.Attribute("keyProperty")
		// The keyProperty is empty, return the result directly.
		if len(keyProperty) == 0 {
			ty := rv.Type()
			// If the keyProperty is empty, try to find from the tag.
			for i := 0; i < ty.NumField(); i++ {
				if autoIncr := ty.Field(i).Tag.Get("autoincr"); autoIncr == "true" {
					field = rv.Field(i)
					break
				}
			}
		} else {
			// try to find the field from the given struct.
			field = rv.FieldByName(keyProperty)
		}

		if !field.IsValid() {
			return nil, fmt.Errorf("the keyProperty %s is not found or not field has the autoincr tag", keyProperty)
		}

		// If the field is not an int, return the result directly.
		if !field.CanInt() {
			return nil, fmt.Errorf("the keyProperty %s is not a int", keyProperty)
		}

		// get the last insert id
		id, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}
		// set the id to the field
		field.SetInt(id)
		return result, nil
	}
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
	// If the cache is nil or the useCache is false, return the result directly.
	if c.cache == nil || stmt.Attribute("useCache") == "false" {
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

		// call the next handler
		result, err = next(ctx, query, args...)
		if err != nil {
			return
		}
		err = c.cache.Set(ctx, cacheKey, result)
		return
	}
}

// ExecContext implements Middleware.
func (c *CacheMiddleware[T]) ExecContext(stmt *Statement, next ExecHandler) ExecHandler {
	// if the cache is enabled and flushCache is not disabled in this statement.
	if stmt.Attribute("flushCache") == "false" || c.cache == nil {
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
