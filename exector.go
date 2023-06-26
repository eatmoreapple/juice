package juice

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/eatmoreapple/juice/cache"
	"reflect"
)

// Executor is an executor of SQL.
type Executor interface {
	// Query executes a query that returns rows, typically a SELECT.
	// The param are the placeholder collection for this query.
	Query(param Param) (*sql.Rows, error)

	// QueryContext executes a query that returns rows, typically a SELECT.
	// The param are the placeholder collection for this query.
	QueryContext(ctx context.Context, param Param) (*sql.Rows, error)

	// Exec executes a query without returning any rows.
	// The param are the placeholder collection for this query.
	Exec(param Param) (sql.Result, error)

	// ExecContext executes a query without returning any rows.
	// The param are the placeholder collection for this query.
	ExecContext(ctx context.Context, param Param) (sql.Result, error)

	// Statement returns the statement of the current executor.
	Statement() *Statement

	// Session returns the session of the current executor.
	Session() Session
}

// inValidExecutor is an invalid executor.
func inValidExecutor(err error) Executor {
	return &executor{err: err}
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

// Query executes the query and returns the result.
func (e *executor) Query(param Param) (*sql.Rows, error) {
	return e.QueryContext(context.Background(), param)
}

// QueryContext executes the query and returns the result.
func (e *executor) QueryContext(ctx context.Context, param Param) (*sql.Rows, error) {
	query, args, err := e.build(param)
	if err != nil {
		return nil, err
	}
	ctx = SessionWithContext(ctx, e.Session())
	ctx = CtxWithParam(ctx, param)
	return e.Statement().QueryHandler()(ctx, query, args...)
}

// Exec executes the query and returns the result.
func (e *executor) Exec(param Param) (sql.Result, error) {
	return e.ExecContext(context.Background(), param)
}

// ExecContext executes the query and returns the result.
func (e *executor) ExecContext(ctx context.Context, param Param) (sql.Result, error) {
	query, args, err := e.build(param)
	if err != nil {
		return nil, err
	}
	ctx = SessionWithContext(ctx, e.Session())

	ctx = CtxWithParam(ctx, param)

	return e.Statement().ExecHandler()(ctx, query, args...)
}

// Statement returns the statement.
func (e *executor) Statement() *Statement {
	return e.statement
}

func (e *executor) Session() Session {
	return e.session
}

// GenericExecutor is a generic executor.
type GenericExecutor[T any] interface {
	// Query executes the query and returns the direct result.
	// The args are for any placeholder parameters in the query.
	Query(param Param) (T, error)

	// QueryContext executes the query and returns the direct result.
	// The args are for any placeholder parameters in the query.
	QueryContext(ctx context.Context, param Param) (T, error)

	// Exec executes a query without returning any rows.
	// The args are for any placeholder parameters in the query.
	Exec(param Param) (sql.Result, error)

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
	cache       cache.Cache
	middlewares GenericMiddlewareGroup[T]
}

// Query executes the query and returns the scanner.
func (e *genericExecutor[T]) Query(p Param) (T, error) {
	return e.QueryContext(context.Background(), p)
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

	if e.cache != nil {
		e.Use(&CacheMiddleware[T]{cache: e.cache})
	}

	// get the cache key
	ctx = SessionWithContext(ctx, e.Session())

	ctx = CtxWithParam(ctx, p)

	// call the middleware
	return e.middlewares.QueryContext(statement, e.queryContext)(ctx, query, args...)
}

func (e *genericExecutor[T]) queryContext(ctx context.Context, query string, args ...any) (result T, err error) {
	statement := e.Statement()

	retMap, err := statement.ResultMap()

	// ErrResultMapNotSet means the result map is not set, use the default result map.
	if err != nil {
		if !errors.Is(err, ErrResultMapNotSet) {
			return
		}
	}

	// try to query the database.
	rows, err := statement.QueryHandler()(ctx, query, args...)
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

// Exec executes the query and returns the result.
func (e *genericExecutor[_]) Exec(p Param) (sql.Result, error) {
	return e.ExecContext(context.Background(), p)
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

var _ GenericExecutor[any] = (*genericExecutor[any])(nil)

// cacheKeyFunc defines the function which is used to generate the cache key.
type cacheKeyFunc func(stmt *Statement, query string, args []any) (string, error)

// CacheKeyFunc is the function which is used to generate the cache key.
// default is the md5 of the query and args.
// reset the CacheKeyFunc variable to change the default behavior.
var CacheKeyFunc cacheKeyFunc = func(stmt *Statement, query string, args []any) (string, error) {
	// only same statement same query same args can get the same cache key
	writer := md5.New()
	writer.Write([]byte(stmt.ID() + query))
	if len(args) > 0 {
		item, err := json.Marshal(args)
		if err != nil {
			return "", err
		}
		writer.Write(item)
	}
	return hex.EncodeToString(writer.Sum(nil)), nil
}
