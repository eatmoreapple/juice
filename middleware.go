package juice

import (
	"context"
	"database/sql"
	"log"
	"strconv"
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
	if cfg := stmt.Configuration(); cfg.Settings.Get("debug") == "false" {
		return false
	}
	return true
}

// TimeoutMiddleware is a middleware that sets the timeout for the sql statement.
type TimeoutMiddleware struct{}

// QueryContext implements Middleware.
// QueryContext will set the timeout for the sql statement.
func (t TimeoutMiddleware) QueryContext(stmt *Statement, next QueryHandler) QueryHandler {
	timeout, ok := t.getTimeout(stmt)
	if !ok {
		return next
	}
	return func(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
		ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
		defer cancel()
		return next(ctx, query, args...)
	}
}

// ExecContext implements Middleware.
// ExecContext will set the timeout for the sql statement.
func (t TimeoutMiddleware) ExecContext(stmt *Statement, next ExecHandler) ExecHandler {
	timeout, ok := t.getTimeout(stmt)
	if !ok {
		return next
	}
	return func(ctx context.Context, query string, args ...any) (sql.Result, error) {
		ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
		defer cancel()
		return next(ctx, query, args...)
	}
}

// getTimeout returns the timeout from the Statement.
func (t TimeoutMiddleware) getTimeout(stmt *Statement) (int64, bool) {
	timeoutAttr := stmt.Attribute("timeout")
	if timeoutAttr == "" {
		return 0, false
	}
	timeout, err := strconv.ParseInt(timeoutAttr, 10, 64)
	if err != nil {
		return 0, false
	}
	if timeout <= 0 {
		return 0, false
	}
	return timeout, true
}
