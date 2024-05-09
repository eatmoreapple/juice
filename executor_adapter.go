package juice

import (
	"context"
	"database/sql"
)

// ExecutorAdapter is an interface for injecting the executor.
type ExecutorAdapter interface {
	// AdapterExecutor injects the executor and returns the new executor.
	AdapterExecutor(Executor) Executor
}

// ExecutorAdapterGroup is a group of executor injectors.
// It implements the ExecutorAdapter interface.
type ExecutorAdapterGroup []ExecutorAdapter

// AdapterExecutor implements the ExecutorAdapter interface.
// It wrapped the executor by the order of the group.
func (eg ExecutorAdapterGroup) AdapterExecutor(e Executor) Executor {
	for _, adapter := range eg {
		e = adapter.AdapterExecutor(e)
	}
	return e
}

// AdapterExecutorFunc is a function type that implements the ExecutorAdapter interface.
type AdapterExecutorFunc func(Executor) Executor

// AdapterExecutor implements the ExecutorAdapter interface.
func (f AdapterExecutorFunc) AdapterExecutor(e Executor) Executor {
	return f(e)
}

// paramCtxInjectorExecutor is an executor that injects the param into the context.
// Which ensures that the param can be used in the middleware.
type paramCtxInjectorExecutor struct {
	Executor
}

// QueryContext executes a query that returns rows, typically a SELECT.
// The param are the placeholder collection for this query.
// The context is injected by the queryContext.
func (e *paramCtxInjectorExecutor) QueryContext(ctx context.Context, param Param) (*sql.Rows, error) {
	ctx = CtxWithParam(ctx, param)
	return e.Executor.QueryContext(ctx, param)
}

// ExecContext executes a query without returning any rows.
// The param are the placeholder collection for this query.
// The context is injected by the execContext.
func (e *paramCtxInjectorExecutor) ExecContext(ctx context.Context, param Param) (sql.Result, error) {
	ctx = CtxWithParam(ctx, param)
	return e.Executor.ExecContext(ctx, param)
}

// NewParamCtxExecutorAdapter returns a new paramCtxInjectorExecutor.
func NewParamCtxExecutorAdapter() ExecutorAdapter {
	return AdapterExecutorFunc(func(e Executor) Executor {
		return &paramCtxInjectorExecutor{Executor: e}
	})
}

// sessionCtxInjectorExecutor is an executor that injects the session into the context.
// Which ensures that the session can be used in the middleware.
type sessionCtxInjectorExecutor struct {
	Executor
}

// QueryContext executes a query that returns rows, typically a SELECT.
// The param are the placeholder collection for this query.
// The context is injected by the sessionContext.
func (e *sessionCtxInjectorExecutor) QueryContext(ctx context.Context, param Param) (*sql.Rows, error) {
	ctx = SessionWithContext(ctx, e.Executor.Session())
	return e.Executor.QueryContext(ctx, param)
}

// ExecContext executes a query without returning any rows.
// The param are the placeholder collection for this query.
// The context is injected by the sessionContext.
func (e *sessionCtxInjectorExecutor) ExecContext(ctx context.Context, param Param) (sql.Result, error) {
	ctx = SessionWithContext(ctx, e.Executor.Session())
	return e.Executor.ExecContext(ctx, param)
}

// NewSessionCtxInjectorExecutorAdapter returns a new sessionCtxInjectorExecutor.
func NewSessionCtxInjectorExecutorAdapter() ExecutorAdapter {
	return AdapterExecutorFunc(func(e Executor) Executor {
		return &sessionCtxInjectorExecutor{Executor: e}
	})
}
