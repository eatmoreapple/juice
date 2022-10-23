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
	QueryContext(stmt *Statement, next QueryHandler) QueryHandler
	// ExecContext wraps the ExecHandler.
	ExecContext(stmt *Statement, next ExecHandler) ExecHandler
}

// MiddlewareGroup is a group of Middleware.
type MiddlewareGroup []Middleware

// QueryContext implements Middleware.
// Call QueryContext will call all the QueryContext of the middlewares in the group.
func (m MiddlewareGroup) QueryContext(stmt *Statement, next QueryHandler) QueryHandler {
	for _, middleware := range m {
		next = middleware.QueryContext(stmt, next)
	}
	return next
}

// ExecContext implements Middleware.
// Call ExecContext will call all the ExecContext of the middlewares in the group.
func (m MiddlewareGroup) ExecContext(stmt *Statement, next ExecHandler) ExecHandler {
	for _, middleware := range m {
		next = middleware.ExecContext(stmt, next)
	}
	return next
}

// logger is a default logger for debug.
var logger = log.New(log.Writer(), "[juice] ", log.Flags())

// DebugMiddleware is a middleware that prints the sql statement and the execution time.
type DebugMiddleware struct{}

// QueryContext implements Middleware.
// QueryContext will print the sql statement and the execution time.
func (m *DebugMiddleware) QueryContext(stmt *Statement, next QueryHandler) QueryHandler {
	if !m.isBugMode(stmt) {
		return next
	}
	// wrapper QueryHandler
	return func(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
		start := time.Now()
		rows, err := next(ctx, query, args...)
		spent := time.Since(start)
		logger.Printf("\x1b[33m[%s]\x1b[0m \x1b[32m %s\x1b[0m \x1b[34m %v\x1b[0m \x1b[31m %v\x1b[0m\n", stmt.Key(), query, args, spent)
		return rows, err
	}
}

// ExecContext implements Middleware.
// ExecContext will print the sql statement and the execution time.
func (m *DebugMiddleware) ExecContext(stmt *Statement, next ExecHandler) ExecHandler {
	if !m.isBugMode(stmt) {
		return next
	}
	// wrapper ExecContext
	return func(ctx context.Context, query string, args ...any) (sql.Result, error) {
		start := time.Now()
		rows, err := next(ctx, query, args...)
		spent := time.Since(start)
		logger.Printf("\x1b[33m[%s]\x1b[0m \x1b[32m %s\x1b[0m \x1b[34m %v\x1b[0m \x1b[31m %v\x1b[0m\n", stmt.Key(), query, args, spent)
		return rows, err
	}
}

// isBugMode returns true if the debug mode is on.
// Default debug mode is on.
// You can turn off the debug mode by setting the debug tag to false in the mapper statement attribute or the configuration.
func (m *DebugMiddleware) isBugMode(stmt *Statement) bool {
	// try to get the bug mode from the Statement
	debug := stmt.Attribute("debug")
	// if the bug mode is not set, try to get the bug mode from the Context
	if debug == "false" {
		return false
	}
	if cfg := stmt.Mapper().Mappers().Configuration(); cfg.Settings.Get("debug") == "false" {
		return false
	}
	return true
}
