package juice

import (
	"context"
	"database/sql"
)

// Session is a wrapper of sql.DB and sql.Tx
type Session interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

// sessionKey is the key of the session in the context.
type sessionKey struct{}

// WithSession returns a new context with the session.
func WithSession(ctx context.Context, session Session) context.Context {
	return context.WithValue(ctx, sessionKey{}, session)
}

// FromContext returns the session from the context.
func FromContext(ctx context.Context) Session {
	session, _ := ctx.Value(sessionKey{}).(Session)
	return session
}
