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
	"fmt"
	"github.com/eatmoreapple/juice/internal/reflectlite"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
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
		logger.Printf("\x1b[33m[%s]\x1b[0m \x1b[32m %s\x1b[0m \x1b[34m %v\x1b[0m \x1b[31m %v\x1b[0m\n", stmt.Name(), query, args, spent)
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
		logger.Printf("\x1b[33m[%s]\x1b[0m \x1b[32m %s\x1b[0m \x1b[34m %v\x1b[0m \x1b[31m %v\x1b[0m\n", stmt.Name(), query, args, spent)
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

		// try to get param from context
		// ParamCtxInjectorExecutor is already set in middlewares, so the param should be in the context.
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
		// NOTE: batch insert does not support useGeneratedKeys yet.
		// TODO: support batch insert useGeneratedKeys.
		if rv.Kind() != reflect.Struct {
			return nil, errors.New("useGeneratedKeys is true, but the param is not a struct pointer")
		}

		var field reflect.Value

		// keyProperty is the name of the field that will be set the generated key.
		keyProperty := stmt.Attribute("keyProperty")

		if len(keyProperty) == 0 {
			// try to find the field by default behavior.
			field = reflectlite.From(rv).FindFieldFromTag("autoincr", "true").Value
		} else {
			keyProperties := strings.Split(keyProperty, ".")
			// try to find the field from the given struct.
			// if isPublic is true, then it means the following keyProperties are the field names.
			// otherwise, the following keyProperties are the tag names.
			isPublic := unicode.IsUpper(rune(keyProperty[0]))

			loopValue := rv

			for i := 0; i < len(keyProperties); i++ {

				value := reflectlite.From(loopValue)

				if ik := value.IndirectKind(); ik != reflect.Struct {
					return nil, fmt.Errorf("expect struct, but got %s", ik)
				}
				// if the keyProperty is public, find the field by name.
				// otherwise, find the field by tag.
				if isPublic {
					loopValue = value.FieldByName(keyProperties[i])
				} else {
					loopValue = value.FindFieldFromTag("column", keyProperties[i]).Value
				}
				// we can not find the field, return directly.
				if !loopValue.IsValid() {
					return nil, fmt.Errorf("the keyProperty %s is not found", keyProperty)
				}
			}
			// reset the field
			field = loopValue
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
