package juice

import (
	"context"
	"database/sql"
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
	id        string
	err       error
	session   Session
	engine    *Engine
	statement Statement
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
	id := e.statement.Namespace() + "." + e.statement.ID()
	return debugForQuery(ctx, e.engine, e.session, id, query, args...)
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
	id := e.statement.Namespace() + "." + e.statement.ID()
	return debugForExec(ctx, e.engine, e.session, id, query, args...)
}

// prepare
func (e *executor) prepare(param interface{}) (query string, args []interface{}, err error) {
	if e.err != nil {
		return "", nil, e.err
	}
	values, err := ParamConvert(param)
	if err != nil {
		return "", nil, err
	}

	translator := e.engine.Driver.Translate()

	return e.statement.Accept(translator, values)
}

// GenericExecutor is a generic executor.
type GenericExecutor[result any] interface {
	Query(param any) Scanner[result]
	QueryContext(ctx context.Context, param any) Scanner[result]
	Exec(param any) (sql.Result, error)
	ExecContext(ctx context.Context, param any) (sql.Result, error)
}

// genericExecutor is a generic executor.
type genericExecutor[result any] struct {
	Executor
}

// Query executes the query and returns the scanner.
func (e *genericExecutor[T]) Query(p any) Scanner[T] {
	return e.QueryContext(context.Background(), p)
}

// QueryContext executes the query and returns the scanner.
func (e *genericExecutor[T]) QueryContext(ctx context.Context, p any) Scanner[T] {
	rows, err := e.Executor.QueryContext(ctx, p)
	return &rowsScanner[T]{rows: rows, err: err}
}

// Exec executes the query and returns the result.
func (e *genericExecutor[result]) Exec(p any) (sql.Result, error) {
	return e.ExecContext(context.Background(), p)
}

// ExecContext executes the query and returns the result.
func (e *genericExecutor[result]) ExecContext(ctx context.Context, p any) (sql.Result, error) {
	return e.Executor.ExecContext(ctx, p)
}
