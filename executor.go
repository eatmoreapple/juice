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

var ErrInvalidExecutor = errors.New("juice: invalid executor")

// GenericExecutor is a generic executor.
type GenericExecutor[T any] interface {
	// QueryContext executes the query and returns the direct result.
	// The args are for any placeholder parameters in the query.
	QueryContext(ctx context.Context, param Param) (T, error)

	// ExecContext executes a query without returning any rows.
	// The args are for any placeholder parameters in the query.
	ExecContext(ctx context.Context, param Param) (sql.Result, error)

	// Statement returns the xmlSQLStatement of the current executor.
	Statement() Statement

	// Session returns the session of the current executor.
	Session() Session

	// Driver returns the driver of the current executor.
	Driver() driver.Driver
}

// Executor defines the interface of the executor.
type Executor GenericExecutor[*sql.Rows]

// invalidExecutor wraps the error who implements the Executor interface.
type invalidExecutor struct {
	_   struct{}
	err error
}

// QueryContext implements the Executor interface.
func (b invalidExecutor) QueryContext(_ context.Context, _ Param) (*sql.Rows, error) {
	return nil, b.err
}

// ExecContext implements the Executor interface.
func (b invalidExecutor) ExecContext(_ context.Context, _ Param) (sql.Result, error) {
	return nil, b.err
}

// Statement implements the Executor interface.
func (b invalidExecutor) Statement() Statement { return nil }

// Session implements the Executor interface.
func (b invalidExecutor) Session() Session { return nil }

func (b invalidExecutor) Driver() driver.Driver { return nil }

// inValidExecutor is an invalid executor.
func inValidExecutor(err error) Executor {
	err = errors.Join(ErrInvalidExecutor, err)
	return &invalidExecutor{err: err}
}

// InValidExecutor returns an invalid executor.
func InValidExecutor() Executor {
	return inValidExecutor(nil)
}

// isInvalidExecutor checks if the executor is a invalidExecutor.
func isInvalidExecutor(e Executor) (*invalidExecutor, bool) {
	exe, ok := e.(*invalidExecutor)
	return exe, ok
}

// ensure that the defaultExecutor implements the Executor interface.
var _ Executor = (*invalidExecutor)(nil)

// executor is an executor of SQL.
type executor struct {
	session     Session
	statement   Statement
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
	queryHandler := CombineQueryHandler(stmt, e.middlewares...)
	return queryHandler(ctx, query, args...)
}

// ExecContext executes the query and returns the result.
func (e *executor) ExecContext(ctx context.Context, param Param) (sql.Result, error) {
	stmt := e.Statement()
	query, args, err := stmt.Build(e.driver.Translator(), param)
	if err != nil {
		return nil, err
	}
	execHandler := CombineExecHandler(stmt, e.middlewares...)
	return execHandler(ctx, query, args...)
}

// Statement returns the xmlSQLStatement.
func (e *executor) Statement() Statement { return e.statement }

// Session returns the session of the executor.
func (e *executor) Session() Session { return e.session }

// Driver returns the driver of the executor.
func (e *executor) Driver() driver.Driver { return e.driver }

// ensure that the executor implements the Executor interface.
var _ Executor = (*executor)(nil)

// genericExecutor is a generic executor.
type genericExecutor[T any] struct {
	Executor
	// extra middlewares for the executor
	middlewares GenericMiddlewareGroup[T]
}

// QueryContext executes the query and returns the scanner.
func (e *genericExecutor[T]) QueryContext(ctx context.Context, p Param) (result T, err error) {
	// check the error of the executor
	if exe, ok := isInvalidExecutor(e.Executor); ok {
		return result, exe.err
	}
	statement := e.Statement()
	// build the query and args
	query, args, err := statement.Build(e.Driver().Translator(), p)
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
	if exe, ok := isInvalidExecutor(e.Executor); ok {
		return ret, exe.err
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
