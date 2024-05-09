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
	"github.com/eatmoreapple/juice/driver"
)

// Executor is an executor of SQL.
type Executor interface {
	// QueryContext executes a query that returns rows, typically a SELECT.
	// The param are the placeholder collection for this query.
	QueryContext(ctx context.Context, param Param) (*sql.Rows, error)

	// ExecContext executes a query without returning any rows.
	// The param are the placeholder collection for this query.
	ExecContext(ctx context.Context, param Param) (sql.Result, error)

	// Statement returns the statement of the current executor.
	Statement() *Statement

	// Session returns the session of the current executor.
	Session() Session

	// Driver returns the driver of the current executor.
	Driver() driver.Driver
}

// inValidExecutor is an invalid executor.
func inValidExecutor(err error) Executor {
	return &badExecutor{err}
}

var (
	// ensure that the defaultExecutor implements the Executor interface.
	_ Executor = (*badExecutor)(nil)

	// ensure that the badExecutor implements the error interface.
	_ error = (*badExecutor)(nil)
)

// badExecutor wraps the error who implements the Executor interface.
type badExecutor struct{ error }

// QueryContext implements the Executor interface.
func (b badExecutor) QueryContext(_ context.Context, _ Param) (*sql.Rows, error) { return nil, b.error }

// ExecContext implements the Executor interface.
func (b badExecutor) ExecContext(_ context.Context, _ Param) (sql.Result, error) { return nil, b.error }

// Statement implements the Executor interface.
func (b badExecutor) Statement() *Statement { return nil }

// Session implements the Executor interface.
func (b badExecutor) Session() Session { return nil }

func (b badExecutor) Driver() driver.Driver { return nil }

// isBadExecutor
func isBadExecutor(e Executor) (*badExecutor, bool) {
	i, ok := e.(*badExecutor)
	return i, ok
}

// executor is an executor of SQL.
type executor struct {
	session     Session
	statement   *Statement
	driver      driver.Driver
	middlewares MiddlewareGroup
}

// QueryContext executes the query and returns the result.
func (e *executor) QueryContext(ctx context.Context, param Param) (*sql.Rows, error) {
	stmt := e.Statement()
	query, args, err := stmt.Build(e.driver.Translator(), param)
	if err != nil {
		return nil, err
	}
	return stmt.QueryHandler(e.middlewares...)(ctx, query, args...)
}

// ExecContext executes the query and returns the result.
func (e *executor) ExecContext(ctx context.Context, param Param) (sql.Result, error) {
	stmt := e.Statement()
	query, args, err := stmt.Build(e.driver.Translator(), param)
	if err != nil {
		return nil, err
	}
	return stmt.ExecHandler(e.middlewares...)(ctx, query, args...)
}

// Statement returns the statement.
func (e *executor) Statement() *Statement {
	return e.statement
}

func (e *executor) Session() Session { return e.session }

func (e *executor) Driver() driver.Driver { return e.driver }

// GenericExecutor is a generic executor.
type GenericExecutor[T any] interface {
	// QueryContext executes the query and returns the direct result.
	// The args are for any placeholder parameters in the query.
	QueryContext(ctx context.Context, param Param) (T, error)

	// ExecContext executes a query without returning any rows.
	// The args are for any placeholder parameters in the query.
	ExecContext(ctx context.Context, param Param) (sql.Result, error)

	// Statement returns the statement of the current executor.
	Statement() *Statement

	// Session returns the session of the current executor.
	Session() Session

	// Use adds a middleware to the current executor.
	// The difference between Engine.Use and Executor.Use is only works for the current executor.
	Use(middlewares ...GenericMiddleware[T])
}

// genericExecutor is a generic executor.
type genericExecutor[T any] struct {
	Executor
	middlewares GenericMiddlewareGroup[T]
}

// QueryContext executes the query and returns the scanner.
func (e *genericExecutor[T]) QueryContext(ctx context.Context, p Param) (result T, err error) {
	// check the error of the executor
	if exe, ok := isBadExecutor(e.Executor); ok {
		return result, exe.error
	}
	statement := e.Statement()
	// build the query and args
	query, args, err := statement.Build(e.Executor.Driver().Translator(), p)
	if err != nil {
		return
	}
	// call the middleware
	return e.middlewares.QueryContext(statement, e.queryContext(p))(ctx, query, args...)
}

func (e *genericExecutor[T]) queryContext(param Param) GenericQueryHandler[T] {
	return func(ctx context.Context, query string, args ...any) (result T, err error) {
		statement := e.Statement()

		retMap, err := statement.ResultMap()

		// ErrResultMapNotSet means the result map is not set, use the default result map.
		if err != nil {
			if !errors.Is(err, ErrResultMapNotSet) {
				return result, err
			}
		}

		// try to query the database.
		rows, err := e.Executor.QueryContext(ctx, param)
		if err != nil {
			return result, err
		}
		defer func() { _ = rows.Close() }()

		return BindWithResultMap[T](rows, retMap)
	}
}

// ExecContext executes the query and returns the result.
func (e *genericExecutor[_]) ExecContext(ctx context.Context, p Param) (ret sql.Result, err error) {
	// check the error of the executor
	if exe, ok := isBadExecutor(e.Executor); ok {
		return ret, exe.error
	}
	return e.Executor.ExecContext(ctx, p)
}

// Use adds a middleware to the current executor.
func (e *genericExecutor[T]) Use(middlewares ...GenericMiddleware[T]) {
	if len(middlewares) == 0 {
		return
	}
	e.middlewares = append(e.middlewares, middlewares...)
}

// ensure genericExecutor implements GenericExecutor.
var _ GenericExecutor[any] = (*genericExecutor[any])(nil)
