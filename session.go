package juice

import (
	"context"
	"database/sql"
	"errors"
)

// Session is a wrapper of sql.DB and sql.Tx
type Session interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

// ErrNoSession is the error that no session found in context.
var ErrNoSession = errors.New("no session found in context")

type sessionKey struct{}

// WithSession returns a new context with the session.
func WithSession(ctx context.Context, sess Session) context.Context {
	return context.WithValue(ctx, sessionKey{}, sess)
}

// SessionFromContext returns the session from the context.
func SessionFromContext(ctx context.Context) Session {
	sess, _ := ctx.Value(sessionKey{}).(Session)
	return sess
}
