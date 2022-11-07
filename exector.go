package juice

import (
	"context"
	"database/sql"
	"reflect"
)

// Executor is an executor of SQL.
type Executor interface {
	Query(param interface{}) (*sql.Rows, error)
	QueryContext(ctx context.Context, param interface{}) (*sql.Rows, error)
	Exec(param interface{}) (sql.Result, error)
	ExecContext(ctx context.Context, param interface{}) (sql.Result, error)
}

// inValidExecutor is an invalid executor.
func inValidExecutor(err error) Executor {
	return &executor{err: err}
}

// executor is an executor of SQL.
type executor struct {
	err       error
	session   Session
	engine    *Engine
	statement *Statement
}

// Query executes the query and returns the result.
func (e *executor) Query(param interface{}) (*sql.Rows, error) {
	return e.QueryContext(context.Background(), param)
}

// QueryContext executes the query and returns the result.
func (e *executor) QueryContext(ctx context.Context, param interface{}) (*sql.Rows, error) {
	query, args, err := e.prepare(param)
	if err != nil {
		return nil, err
	}
	middlewares := e.engine.middlewares
	stmt := e.statement
	ctx = WithSession(ctx, e.session)
	return middlewares.QueryContext(stmt, sessionQueryHandler())(ctx, query, args...)
}

// Exec executes the query and returns the result.
func (e *executor) Exec(param interface{}) (sql.Result, error) {
	return e.ExecContext(context.Background(), param)
}

// ExecContext executes the query and returns the result.
func (e *executor) ExecContext(ctx context.Context, param interface{}) (sql.Result, error) {
	query, args, err := e.prepare(param)
	if err != nil {
		return nil, err
	}
	middlewares := e.engine.middlewares
	stmt := e.statement
	ctx = WithSession(ctx, e.session)
	return middlewares.ExecContext(stmt, sessionExecHandler())(ctx, query, args...)
}

// prepare
func (e *executor) prepare(param interface{}) (query string, args []interface{}, err error) {
	if e.err != nil {
		return "", nil, e.err
	}
	values, err := ParamConvert(param, e.statement.Attribute("paramName"))
	if err != nil {
		return "", nil, err
	}
	translator := e.engine.Driver.Translate()
	query, args, err = e.statement.Accept(translator, values)
	if err != nil {
		return "", nil, err
	}
	if len(query) == 0 {
		return "", nil, ErrEmptyQuery
	}
	return query, args, nil
}

// GenericExecutor is a generic executor.
type GenericExecutor[result any] interface {
	Query(param any) (result, error)
	QueryContext(ctx context.Context, param any) (result, error)
	Exec(param any) (sql.Result, error)
	ExecContext(ctx context.Context, param any) (sql.Result, error)
}

// genericExecutor is a generic executor.
type genericExecutor[result any] struct {
	Executor
}

// Query executes the query and returns the scanner.
func (e *genericExecutor[T]) Query(p any) (T, error) {
	return e.QueryContext(context.Background(), p)
}

// QueryContext executes the query and returns the scanner.
func (e *genericExecutor[T]) QueryContext(ctx context.Context, p any) (result T, err error) {
	rows, err := e.Executor.QueryContext(ctx, p)
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()
	rv := reflect.ValueOf(result)
	if rv.Kind() == reflect.Ptr {
		result = reflect.New(rv.Type().Elem()).Interface().(T)
	}
	err = Bind(rows, &result)
	return
}

// Exec executes the query and returns the result.
func (e *genericExecutor[result]) Exec(p any) (sql.Result, error) {
	return e.ExecContext(context.Background(), p)
}

// ExecContext executes the query and returns the result.
func (e *genericExecutor[result]) ExecContext(ctx context.Context, p any) (sql.Result, error) {
	return e.Executor.ExecContext(ctx, p)
}

var _ GenericExecutor[interface{}] = (*genericExecutor[interface{}])(nil)
