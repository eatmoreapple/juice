package juice

import (
	"context"
	"database/sql"
)

// Session is a wrapper of sql.DB and sql.Tx
type Session interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}
