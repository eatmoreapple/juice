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
	"reflect"
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
}

// ParamCtxInjectorExecutor is an executor that injects the param into the context.
// Which ensures that the param can be used in the middleware.
type ParamCtxInjectorExecutor struct {
	Executor
}

// QueryContext executes a query that returns rows, typically a SELECT.
// The param are the placeholder collection for this query.
// The context is injected by the queryContext.
func (e *ParamCtxInjectorExecutor) QueryContext(ctx context.Context, param Param) (*sql.Rows, error) {
	ctx = CtxWithParam(ctx, param)
	return e.Executor.QueryContext(ctx, param)
}

// ExecContext executes a query without returning any rows.
// The param are the placeholder collection for this query.
// The context is injected by the execContext.
func (e *ParamCtxInjectorExecutor) ExecContext(ctx context.Context, param Param) (sql.Result, error) {
	ctx = CtxWithParam(ctx, param)
	return e.Executor.ExecContext(ctx, param)
}

// SessionCtxInjectorExecutor is an executor that injects the session into the context.
// Which ensures that the session can be used in the middleware.
type SessionCtxInjectorExecutor struct {
	Executor
}

// QueryContext executes a query that returns rows, typically a SELECT.
// The param are the placeholder collection for this query.
// The context is injected by the sessionContext.
func (e *SessionCtxInjectorExecutor) QueryContext(ctx context.Context, param Param) (*sql.Rows, error) {
	ctx = SessionWithContext(ctx, e.Executor.Session())
	return e.Executor.QueryContext(ctx, param)
}

// ExecContext executes a query without returning any rows.
// The param are the placeholder collection for this query.
// The context is injected by the sessionContext.
func (e *SessionCtxInjectorExecutor) ExecContext(ctx context.Context, param Param) (sql.Result, error) {
	ctx = SessionWithContext(ctx, e.Executor.Session())
	return e.Executor.ExecContext(ctx, param)
}

// inValidExecutor is an invalid executor.
func inValidExecutor(err error) Executor {
	return &executor{err: err}
}

func ctxInjectExecutor(executor Executor) Executor {
	sessionCtxInjectorExecutor := &SessionCtxInjectorExecutor{Executor: executor}
	paramCtxInjectorExecutor := &ParamCtxInjectorExecutor{Executor: sessionCtxInjectorExecutor}
	return paramCtxInjectorExecutor
}

// executor is an executor of SQL.
type executor struct {
	session   Session
	statement *Statement
	err       error
}

// build builds the query and args.
func (e *executor) build(param Param) (query string, args []any, err error) {
	if e.err != nil {
		return "", nil, e.err
	}
	return e.Statement().Build(param)
}

// QueryContext executes the query and returns the result.
func (e *executor) QueryContext(ctx context.Context, param Param) (*sql.Rows, error) {
	query, args, err := e.build(param)
	if err != nil {
		return nil, err
	}
	return e.Statement().QueryHandler()(ctx, query, args...)
}

// ExecContext executes the query and returns the result.
func (e *executor) ExecContext(ctx context.Context, param Param) (sql.Result, error) {
	query, args, err := e.build(param)
	if err != nil {
		return nil, err
	}
	return e.Statement().ExecHandler()(ctx, query, args...)
}

// Statement returns the statement.
func (e *executor) Statement() *Statement {
	return e.statement
}

func (e *executor) Session() Session {
	return e.session
}

func (e *executor) IsValid() (bool, error) {
	return e.err == nil, e.err
}

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
	if exe, ok := e.Executor.(*executor); ok && exe.err != nil {
		return result, exe.err
	}
	statement := e.Statement()
	// build the query and args
	query, args, err := statement.Build(p)
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
				return
			}
		}

		// try to query the database.
		rows, err := e.Executor.QueryContext(ctx, param)
		if err != nil {
			return
		}
		defer func() { _ = rows.Close() }()

		// ptr is the pointer of the result, it is the destination of the binding.
		var ptr any = &result

		rv := reflect.ValueOf(result)

		// if the result is a pointer, create a new instance of the element.
		// you'd better not use a nil pointer as the result.
		if rv.Kind() == reflect.Ptr {
			result = reflect.New(rv.Type().Elem()).Interface().(T)
			ptr = result
		}

		err = BindWithResultMap(rows, ptr, retMap)
		return
	}
}

// ExecContext executes the query and returns the result.
func (e *genericExecutor[_]) ExecContext(ctx context.Context, p Param) (ret sql.Result, err error) {
	// check the error of the executor
	if exe, ok := e.Executor.(*executor); ok && exe.err != nil {
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
