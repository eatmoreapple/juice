/*
Copyright 2023 eatmoreapple

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package juice defines the handler of the query and exec.

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

// sessionQueryHandler is the default QueryHandler.
// It will get the session from the context.
// And use the session to query the database.
func sessionQueryHandler(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	sess, err := session.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	return sess.QueryContext(ctx, query, args...)
}

// sessionExecHandler is the default ExecHandler.
// It will get the session from the context.
// And use the session to exec the database.
func sessionExecHandler(ctx context.Context, query string, args ...any) (sql.Result, error) {
	sess, err := session.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	return sess.ExecContext(ctx, query, args...)
}

// GenericQueryHandler defines the handler of the generic query.
type GenericQueryHandler[T any] func(ctx context.Context, query string, args ...any) (T, error)

// CombineQueryHandler will combine the middlewares and the default QueryHandler.
// If the middlewares is empty, it will return the default QueryHandler.
func CombineQueryHandler(stmt Statement, middlewares ...Middleware) QueryHandler {
	if len(middlewares) > 0 {
		group := MiddlewareGroup(middlewares)
		return group.QueryContext(stmt, sessionQueryHandler)
	}
	return sessionQueryHandler
}

// CombineExecHandler will combine the middlewares and the default ExecHandler.
// If the middlewares is empty, it will return the default ExecHandler.
func CombineExecHandler(stmt Statement, middlewares ...Middleware) ExecHandler {
	if len(middlewares) > 0 {
		group := MiddlewareGroup(middlewares)
		return group.ExecContext(stmt, sessionExecHandler)
	}
	return sessionExecHandler
}
