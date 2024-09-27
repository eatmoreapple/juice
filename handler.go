package juice

import (
	"context"
	"database/sql"

	"github.com/eatmoreapple/juice/session"
)

// QueryHandler defines the handler of the query.
type QueryHandler func(ctx context.Context, query string, args ...any) (*sql.Rows, error)

// ExecHandler defines the handler of the exec.
type ExecHandler func(ctx context.Context, query string, args ...any) (sql.Result, error)

func sessionQueryHandler() QueryHandler {
	return func(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
		sess := session.FromContext(ctx)
		if sess == nil {
			return nil, session.ErrNoSession
		}
		return sess.QueryContext(ctx, query, args...)
	}
}

func sessionExecHandler() ExecHandler {
	return func(ctx context.Context, query string, args ...any) (sql.Result, error) {
		sess := session.FromContext(ctx)
		if sess == nil {
			return nil, session.ErrNoSession
		}
		return sess.ExecContext(ctx, query, args...)
	}
}

// GenericQueryHandler defines the handler of the generic query.
type GenericQueryHandler[T any] func(ctx context.Context, query string, args ...any) (T, error)

func CombineQueryHandler(stmt Statement, middlewares ...Middleware) QueryHandler {
	next := sessionQueryHandler()
	if len(middlewares) > 0 {
		group := MiddlewareGroup(middlewares)
		return group.QueryContext(stmt, next)
	}
	return next
}

func CombineExecHandler(stmt Statement, middlewares ...Middleware) ExecHandler {
	next := sessionExecHandler()
	if len(middlewares) > 0 {
		group := MiddlewareGroup(middlewares)
		return group.ExecContext(stmt, next)
	}
	return next
}
