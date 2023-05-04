package juice

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
)

// Executor is an executor of SQL.
type Executor interface {
	Query(param any) (*sql.Rows, error)
	QueryContext(ctx context.Context, param any) (*sql.Rows, error)
	Exec(param any) (sql.Result, error)
	ExecContext(ctx context.Context, param any) (sql.Result, error)
	Statement() *Statement
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
func (e *executor) Query(param any) (*sql.Rows, error) {
	return e.QueryContext(context.Background(), param)
}

// QueryContext executes the query and returns the result.
func (e *executor) QueryContext(ctx context.Context, param any) (*sql.Rows, error) {
	query, args, err := e.prepare(param)
	if err != nil {
		return nil, err
	}
	middlewares := e.engine.middlewares
	stmt := e.statement
	ctx = SessionWithContext(ctx, e.session)
	return middlewares.QueryContext(stmt, sessionQueryHandler())(ctx, query, args...)
}

// Exec executes the query and returns the result.
func (e *executor) Exec(param any) (sql.Result, error) {
	return e.ExecContext(context.Background(), param)
}

// ExecContext executes the query and returns the result.
func (e *executor) ExecContext(ctx context.Context, param any) (sql.Result, error) {
	query, args, err := e.prepare(param)
	if err != nil {
		return nil, err
	}
	middlewares := e.engine.middlewares
	stmt := e.statement
	ctx = SessionWithContext(ctx, e.session)
	return middlewares.ExecContext(stmt, sessionExecHandler())(ctx, query, args...)
}

// Statement returns the statement.
func (e *executor) Statement() *Statement {
	return e.statement
}

// prepare
func (e *executor) prepare(param any) (query string, args []any, err error) {
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

	retMap, err := e.Executor.Statement().ResultMap()

	// set but not found
	if err != nil {
		if !errors.Is(err, ErrResultMapNotSet) {
			return result, err
		}
	}

	rv := reflect.ValueOf(result)

	if rv.Kind() == reflect.Ptr {
		// if T is a pointer, then set prt to T
		value := reflect.New(rv.Type().Elem()).Interface().(T)
		if err = BindWithResultMap(rows, value, retMap); err != nil {
			// if bind failed, then return the original value
			// result is a zero value
			return result, err
		}
		// if bind success, then return the new value
		return value, nil
	}
	// bind the result to the pointer
	err = BindWithResultMap(rows, &result, retMap)
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

var _ GenericExecutor[any] = (*genericExecutor[any])(nil)

// BinderExecutor is a binder executor.
// It is used to bind the result to the given value.
type BinderExecutor interface {
	Query(param any) (Binder, error)
	QueryContext(ctx context.Context, param any) (Binder, error)
	Exec(param any) (sql.Result, error)
	ExecContext(ctx context.Context, param any) (sql.Result, error)
}

// binderExecutor is a binder executor.
// binderExecutor implements the BinderExecutor interface.
type binderExecutor struct {
	Executor
}

// Query executes the query and returns the scanner.
func (b *binderExecutor) Query(param any) (Binder, error) {
	return b.QueryContext(context.Background(), param)
}

// QueryContext executes the query and returns the scanner.
func (b *binderExecutor) QueryContext(ctx context.Context, param any) (Binder, error) {
	rows, err := b.Executor.QueryContext(ctx, param)
	if err != nil {
		return nil, err
	}
	retMap, err := b.Executor.Statement().ResultMap()
	if err != nil && !errors.Is(err, ErrResultMapNotSet) {
		return nil, err
	}
	return &rowsBinder{rows: rows, mapper: retMap}, nil
}

var _ BinderExecutor = (*binderExecutor)(nil)
