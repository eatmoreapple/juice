package juice

import (
	"context"
	"database/sql"
)

// QueryHandler defines the handler of the query.
type QueryHandler func(ctx context.Context, query string, args ...any) (*sql.Rows, error)

// ExecHandler defines the handler of the exec.
type ExecHandler func(ctx context.Context, query string, args ...any) (sql.Result, error)
