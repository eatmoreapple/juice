package juice

import (
	"context"
	"database/sql"
)

// QueryHandler defines the handler of the query.
type QueryHandler func(ctx context.Context, query string, args ...any) (*sql.Rows, error)

// ExecHandler defines the handler of the exec.
type ExecHandler func(ctx context.Context, query string, args ...any) (sql.Result, error)

func sessionQueryHandler() QueryHandler {
	return func(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
		sess := SessionFromContext(ctx)
		if sess == nil {
			return nil, ErrNoSession
		}
		return sess.QueryContext(ctx, query, args...)
	}
}

func sessionExecHandler() ExecHandler {
	return func(ctx context.Context, query string, args ...any) (sql.Result, error) {
		sess := SessionFromContext(ctx)
		if sess == nil {
			return nil, ErrNoSession
		}
		return sess.ExecContext(ctx, query, args...)
	}
}

// GenericQueryHandler defines the handler of the generic query.
type GenericQueryHandler[T any] func(ctx context.Context, query string, args ...any) (T, error)
