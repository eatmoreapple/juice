package juice

import (
	"context"
	"database/sql"
)

type StatementExecutor interface {
	Statement(v interface{}) Executor
}

// Executor is an executor of SQL.
type Executor interface {
	Query(param interface{}) (*sql.Rows, error)
	QueryContext(ctx context.Context, param interface{}) (*sql.Rows, error)
	Exec(param interface{}) (sql.Result, error)
	ExecContext(ctx context.Context, param interface{}) (sql.Result, error)
}

type executor struct {
	id        string
	err       error
	session   Session
	engine    *Engine
	statement Statement
}

func (e *executor) Query(param interface{}) (*sql.Rows, error) {
	return e.QueryContext(context.Background(), param)
}

func (e *executor) QueryContext(ctx context.Context, param interface{}) (*sql.Rows, error) {
	query, args, err := e.prepare(param)
	if err != nil {
		return nil, err
	}
	return e.session.QueryContext(ctx, query, args...)
}

func (e *executor) Exec(param interface{}) (sql.Result, error) {
	return e.ExecContext(context.Background(), param)
}

func (e *executor) ExecContext(ctx context.Context, param interface{}) (sql.Result, error) {
	query, args, err := e.prepare(param)
	if err != nil {
		return nil, err
	}
	return e.session.ExecContext(ctx, query, args...)
}

func (e *executor) prepare(param interface{}) (query string, args []interface{}, err error) {
	if e.err != nil {
		return "", nil, e.err
	}
	values, err := ParamConvert(param)
	if err != nil {
		return "", nil, err
	}

	translator := e.engine.Driver.Translate()

	query, args, err = e.statement.Accept(translator, values)
	if err != nil {
		return "", nil, err
	}

	if e.engine.Logger != nil {
		e.engine.Logger.Printf("[%s] query: {%s} args: %v", e.statement.Namespace()+"."+e.statement.ID(), query, args)
	}
	return query, args, nil
}

type TxMapperExecutor interface {
	StatementExecutor
	Commit() error
	Rollback() error
}

type txStatement struct {
	engine *Engine
	tx     *sql.Tx
	err    error
}

func (t *txStatement) Statement(v interface{}) Executor {
	stat, err := t.engine.getMapperStatement(v)
	return &executor{err: err, engine: t.engine, statement: stat, session: t.tx}
}

func (t *txStatement) Commit() error {
	if t.err != nil {
		return t.err
	}
	return t.tx.Commit()
}

func (t *txStatement) Rollback() error {
	if t.err != nil {
		return t.err
	}
	return t.tx.Rollback()
}

type GenericMapperExecutor[result, param any] interface {
	Statement(v any) GenericExecutor[result, param]
}

type GenericExecutor[result, param any] interface {
	Query(param param) Scanner[result]
	QueryContext(ctx context.Context, param param) Scanner[result]
	Exec(param param) (sql.Result, error)
	ExecContext(ctx context.Context, param param) (sql.Result, error)
}

type genericExecutor[result, param any] struct {
	Executor
}

func (e *genericExecutor[T, param]) Query(p param) Scanner[T] {
	rows, err := e.Executor.Query(p)
	return &rowsScanner[T]{rows: rows, err: err}
}

func (e *genericExecutor[result, param]) QueryContext(ctx context.Context, p param) Scanner[result] {
	rows, err := e.Executor.QueryContext(ctx, p)
	return &rowsScanner[result]{rows: rows, err: err}
}

func (e *genericExecutor[result, param]) Exec(p param) (sql.Result, error) {
	return e.Executor.Exec(p)
}

func (e *genericExecutor[result, param]) ExecContext(ctx context.Context, p param) (sql.Result, error) {
	return e.Executor.ExecContext(ctx, p)
}

type TxGenericStatementExecutor[result, param any] interface {
	GenericMapperExecutor[result, param]
	Commit() error
	Rollback() error
}

type txGenericStatement[result, param any] struct {
	TxMapperExecutor
}

func (t *txGenericStatement[result, param]) Statement(v any) GenericExecutor[result, param] {
	return &genericExecutor[result, param]{Executor: t.TxMapperExecutor.Statement(v)}
}
