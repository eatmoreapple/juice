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
	"errors"
	"log"
	"reflect"
	"strconv"
	"time"
)

// Middleware is a wrapper of QueryHandler and ExecHandler.
type Middleware interface {
	// QueryContext wraps the QueryHandler.
	QueryContext(stmt Statement, next QueryHandler) QueryHandler
	// ExecContext wraps the ExecHandler.
	ExecContext(stmt Statement, next ExecHandler) ExecHandler
}

// ensure MiddlewareGroup implements Middleware.
var _ Middleware = MiddlewareGroup(nil) // compile time check

// MiddlewareGroup is a group of Middleware.
type MiddlewareGroup []Middleware

// QueryContext implements Middleware.
// Call QueryContext will call all the QueryContext of the middlewares in the group.
func (m MiddlewareGroup) QueryContext(stmt Statement, next QueryHandler) QueryHandler {
	for _, middleware := range m {
		next = middleware.QueryContext(stmt, next)
	}
	return next
}

// ExecContext implements Middleware.
// Call ExecContext will call all the ExecContext of the middlewares in the group.
func (m MiddlewareGroup) ExecContext(stmt Statement, next ExecHandler) ExecHandler {
	for _, middleware := range m {
		next = middleware.ExecContext(stmt, next)
	}
	return next
}

// logger is a default logger for debug.
var logger = log.New(log.Writer(), "[juice] ", log.Flags())

// ensure DebugMiddleware implements Middleware.
var _ Middleware = (*DebugMiddleware)(nil) // compile time check

// DebugMiddleware is a middleware that prints the sql xmlSQLStatement and the execution time.
type DebugMiddleware struct{}

// QueryContext implements Middleware.
// QueryContext will print the sql xmlSQLStatement and the execution time.
func (m *DebugMiddleware) QueryContext(stmt Statement, next QueryHandler) QueryHandler {
	if !m.isDeBugMode(stmt) {
		return next
	}
	// wrapper QueryHandler
	return func(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
		start := time.Now()
		rows, err := next(ctx, query, args...)
		spent := time.Since(start)
		logger.Printf("\x1b[33m[%s]\x1b[0m \x1b[32m %s\x1b[0m \x1b[38m %v\x1b[0m \x1b[31m %v\x1b[0m\n", stmt.Name(), query, args, spent)
		return rows, err
	}
}

// ExecContext implements Middleware.
// ExecContext will print the sql xmlSQLStatement and the execution time.
func (m *DebugMiddleware) ExecContext(stmt Statement, next ExecHandler) ExecHandler {
	if !m.isDeBugMode(stmt) {
		return next
	}
	// wrapper ExecContext
	return func(ctx context.Context, query string, args ...any) (sql.Result, error) {
		start := time.Now()
		rows, err := next(ctx, query, args...)
		spent := time.Since(start)
		logger.Printf("\x1b[33m[%s]\x1b[0m \x1b[32m %s\x1b[0m \x1b[38m %v\x1b[0m \x1b[31m %v\x1b[0m\n", stmt.Name(), query, args, spent)
		return rows, err
	}
}

// isDeBugMode returns true if the debug mode is on.
// Default debug mode is on.
// You can turn off the debug mode by setting the debug tag to false in the mapper xmlSQLStatement attribute or the configuration.
func (m *DebugMiddleware) isDeBugMode(stmt Statement) bool {
	// try to one the bug mode from the xmlSQLStatement
	debug := stmt.Attribute("debug")
	// if the bug mode is not set, try to one the bug mode from the Context
	if debug == "false" {
		return false
	}
	if cfg := stmt.Configuration(); cfg.Settings().Get("debug") == "false" {
		return false
	}
	return true
}

// ensure TimeoutMiddleware implements Middleware
var _ Middleware = (*TimeoutMiddleware)(nil) // compile time check

// TimeoutMiddleware is a middleware that sets the timeout for the sql xmlSQLStatement.
type TimeoutMiddleware struct{}

// QueryContext implements Middleware.
// QueryContext will set the timeout for the sql xmlSQLStatement.
func (t TimeoutMiddleware) QueryContext(stmt Statement, next QueryHandler) QueryHandler {
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
// ExecContext will set the timeout for the sql xmlSQLStatement.
func (t TimeoutMiddleware) ExecContext(stmt Statement, next ExecHandler) ExecHandler {
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

// getTimeout returns the timeout from the xmlSQLStatement.
func (t TimeoutMiddleware) getTimeout(stmt Statement) (timeout int64) {
	timeoutAttr := stmt.Attribute("timeout")
	if timeoutAttr == "" {
		return
	}
	timeout, _ = strconv.ParseInt(timeoutAttr, 10, 64)
	return
}

// ensure useGeneratedKeysMiddleware implements Middleware
var _ Middleware = (*useGeneratedKeysMiddleware)(nil) // compile time check

// errStructPointerOrSliceArrayRequired is an error that the param is not a struct pointer or a slice array type.
var errStructPointerOrSliceArrayRequired = errors.New(
	"useGeneratedKeys is true, but the param is not a struct pointer or a slice array type",
)

// useGeneratedKeysMiddleware is a middleware that set the last insert id to the struct.
type useGeneratedKeysMiddleware struct{}

// QueryContext implements Middleware.
// return the result directly and do nothing.
func (m *useGeneratedKeysMiddleware) QueryContext(_ Statement, next QueryHandler) QueryHandler {
	return next
}

// ExecContext implements Middleware.
// ExecContext will set the last insert id to the struct.
func (m *useGeneratedKeysMiddleware) ExecContext(stmt Statement, next ExecHandler) ExecHandler {
	if !(stmt.Action() == Insert) {
		return next
	}
	const _useGeneratedKeys = "useGeneratedKeys"
	// If the useGeneratedKeys is not set or false, return the result directly.
	useGeneratedKeys := stmt.Attribute(_useGeneratedKeys) == "true" ||
		// If the useGeneratedKeys is not set, but the global useGeneratedKeys is set and true.
		stmt.Configuration().Settings().Get(_useGeneratedKeys) == "true"

	if !useGeneratedKeys {
		return next
	}
	return func(ctx context.Context, query string, args ...any) (sql.Result, error) {
		result, err := next(ctx, query, args...)
		if err != nil {
			return nil, err
		}

		id, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}
		// try to get param from context
		// ParamCtxInjectorExecutor is already set in middlewares, so the param should be in the context.
		param := ParamFromContext(ctx)

		if param == nil {
			return nil, errors.New("useGeneratedKeys is true, but the param is nil")
		}

		// checkout the input param
		rv := reflect.ValueOf(param)

		keyProperty := stmt.Attribute("keyProperty")

		var keyGenerator selectKeyGenerator

		switch reflect.Indirect(rv).Kind() {
		case reflect.Struct:
			keyGenerator = &singleKeyGenerator{
				keyProperty: keyProperty,
				id:          id,
			}
		case reflect.Array, reflect.Slice:
			// try to get the keyIncrement from the xmlSQLStatement
			// if the keyIncrement is not set or invalid, use the default value 1
			keyIncrementValue := stmt.Attribute("keyIncrement")
			keyIncrement, _ := strconv.ParseInt(keyIncrementValue, 10, 64)
			if keyIncrement == 0 {
				keyIncrement = 1
			}
			keyGenerator = &batchKeyGenerator{
				keyProperty:  keyProperty,
				id:           id,
				keyIncrement: keyIncrement,
			}
		default:
			return nil, errStructPointerOrSliceArrayRequired
		}
		if err = keyGenerator.GenerateKeyTo(rv); err != nil {
			return nil, err
		}
		return result, nil
	}
}
