package juice

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// Middleware is a wrapper of QueryHandler and ExecHandler.
type Middleware interface {
	// QueryContext wraps the QueryHandler.
	QueryContext(ctx *Context, next QueryHandler) QueryHandler
	// ExecContext wraps the ExecHandler.
	ExecContext(ctx *Context, next ExecHandler) ExecHandler
}

// MiddlewareGroup is a group of Middleware.
type MiddlewareGroup []Middleware

// QueryContext implements Middleware.
// Call QueryContext will call all the QueryContext of the middlewares in the group.
func (m MiddlewareGroup) QueryContext(c *Context, next QueryHandler) QueryHandler {
	for _, middleware := range m {
		next = middleware.QueryContext(c, next)
	}
	return next
}

// ExecContext implements Middleware.
// Call ExecContext will call all the ExecContext of the middlewares in the group.
func (m MiddlewareGroup) ExecContext(c *Context, next ExecHandler) ExecHandler {
	for _, middleware := range m {
		next = middleware.ExecContext(c, next)
	}
	return next
}

// logger is a default logger for debug.
var logger = log.New(log.Writer(), "[juice] ", log.Flags())

// DebugMiddleware is a middleware that prints the sql statement and the execution time.
type DebugMiddleware struct{}

// QueryContext implements Middleware.
// QueryContext will print the sql statement and the execution time.
func (m *DebugMiddleware) QueryContext(c *Context, next QueryHandler) QueryHandler {
	if !m.isBugMode(c) {
		return next
	}
	// wrapper QueryHandler
	return func(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
		start := time.Now()
		rows, err := next(ctx, query, args...)
		spent := time.Since(start)
		id := c.Statement.Key()
		logger.Printf("\x1b[33m[%s]\x1b[0m \x1b[32m %s\x1b[0m \x1b[34m %v\x1b[0m \x1b[31m %v\x1b[0m\n", id, query, args, spent)
		return rows, err
	}
}

// ExecContext implements Middleware.
// ExecContext will print the sql statement and the execution time.
func (m *DebugMiddleware) ExecContext(c *Context, next ExecHandler) ExecHandler {
	if !m.isBugMode(c) {
		return next
	}
	// wrapper ExecContext
	return func(ctx context.Context, query string, args ...any) (sql.Result, error) {
		start := time.Now()
		rows, err := next(ctx, query, args...)
		spent := time.Since(start)
		id := c.Statement.Key()
		logger.Printf("\x1b[33m[%s]\x1b[0m \x1b[32m %s\x1b[0m \x1b[34m %v\x1b[0m \x1b[31m %v\x1b[0m\n", id, query, args, spent)
		return rows, err
	}
}

// isBugMode returns true if the debug mode is on.
func (m *DebugMiddleware) isBugMode(c *Context) bool {
	// try to get the bug mode from the Statement
	debug := c.Statement.Attribute("debug")
	// if the bug mode is not set, try to get the bug mode from the Context
	if debug == "false" {
		return false
	}
	return debug == "true" || c.Configuration.Settings.Get("debug") == "true"
}
