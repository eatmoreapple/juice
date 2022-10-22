package juice

import (
	"context"
	"database/sql"
	"log"
	"time"
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
	if e.engine.configuration.Settings.Debug() {
		return debugForQuery(ctx, e.session, e.statement.Key(), query, args...)
	}
	return e.session.QueryContext(ctx, query, args...)
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
	if e.engine.configuration.Settings.Debug() {
		return debugForExec(ctx, e.session, e.statement.Key(), query, args...)
	}
	return e.session.ExecContext(ctx, query, args...)
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

var _ GenericExecutor[interface{}] = (*genericExecutor[interface{}])(nil)

// logger is a default logger for debug.
var logger = log.New(log.Writer(), "[juice] ", log.Flags())

// debugForQuery executes the query and logs the result.
// If debug is enabled, it will log the query and the arguments.
// If debug is disabled, it will execute the query directly.
func debugForQuery(ctx context.Context, session Session, id string, query string, args ...any) (*sql.Rows, error) {
	start := time.Now()
	rows, err := session.QueryContext(ctx, query, args...)
	spent := time.Since(start)
	logger.Printf("\x1b[33m[%s]\x1b[0m \x1b[32m %s\x1b[0m \x1b[34m %v\x1b[0m \x1b[31m %v\x1b[0m\n", id, query, args, spent)
	return rows, err
}

// debugForExec executes the query and logs the result.
// If debug is enabled, it will log the query and the arguments.
// If debug is disabled, it will execute the query directly.
func debugForExec(ctx context.Context, session Session, id string, query string, args ...any) (sql.Result, error) {
	start := time.Now()
	rows, err := session.ExecContext(ctx, query, args...)
	spent := time.Since(start)
	logger.Printf("\x1b[33m[%s]\x1b[0m \x1b[32m %s\x1b[0m \x1b[34m %v\x1b[0m \x1b[31m %v\x1b[0m\n", id, query, args, spent)
	return rows, err
}
