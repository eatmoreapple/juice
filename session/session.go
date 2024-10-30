/*
Copyright 2024 eatmoreapple

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

package session

import (
	"context"
	"database/sql"
)

// Session is a wrapper of sql.DB and sql.Tx
type Session interface {
	// QueryContext executes the query and returns the direct result.
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)

	// ExecContext executes a query without returning any rows.
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)

	// PrepareContext creates a prepared statement for later queries or executions.
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

var (
	// ensure that the sql.DB implements the Session interface.
	_ Session = (*sql.DB)(nil)

	// ensure that the sql.Tx implements the Session interface.
	_ Session = (*sql.Tx)(nil)
)
