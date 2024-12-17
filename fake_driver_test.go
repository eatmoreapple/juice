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

// testing fake driver

package juice

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

func init() {
	sql.Register("fake", &fakeDriver{})
}

// fakeDriver is a fake database driver for testing.
type fakeDriver struct{}

// Open returns a new fake connection.
func (d *fakeDriver) Open(name string) (driver.Conn, error) {
	return &fakeConn{}, nil
}

// fakeConn is a fake database connection.
type fakeConn struct {
	closed bool
}

func (c *fakeConn) Prepare(query string) (driver.Stmt, error) {
	if c.closed {
		return nil, driver.ErrBadConn
	}
	return &fakeStmt{query: query}, nil
}

func (c *fakeConn) Close() error {
	c.closed = true
	return nil
}

func (c *fakeConn) Begin() (driver.Tx, error) {
	if c.closed {
		return nil, driver.ErrBadConn
	}
	return &fakeTx{}, nil
}

// fakeStmt is a fake prepared statement.
type fakeStmt struct {
	query  string
	closed bool
}

func (s *fakeStmt) Close() error {
	s.closed = true
	return nil
}

func (s *fakeStmt) NumInput() int {
	return -1 // driver doesn't know how many parameters there are
}

func (s *fakeStmt) Exec(args []driver.Value) (driver.Result, error) {
	if s.closed {
		return nil, driver.ErrBadConn
	}
	return &fakeResult{}, nil
}

func (s *fakeStmt) Query(args []driver.Value) (driver.Rows, error) {
	if s.closed {
		return nil, driver.ErrBadConn
	}

	maxRows := 3 // default to 3 rows
	// Check if the query contains LIMIT
	if strings.Contains(strings.ToLower(s.query), "limit") && len(args) > 0 {
		if limit, ok := args[0].(int64); ok && limit > 0 {
			maxRows = int(limit)
		}
	}

	return &fakeRows{maxRows: maxRows}, nil
}

// fakeResult is a fake result set.
type fakeResult struct{}

func (r *fakeResult) LastInsertId() (int64, error) {
	return 1, nil
}

func (r *fakeResult) RowsAffected() (int64, error) {
	return 1, nil
}

// fakeRows is a fake rows implementation.
type fakeRows struct {
	currentRow int
	closed     bool
	maxRows    int
}

func (r *fakeRows) Columns() []string {
	return []string{"id", "name", "created_at"}
}

func (r *fakeRows) Close() error {
	r.closed = true
	return nil
}

func (r *fakeRows) Next(dest []driver.Value) error {
	if r.closed {
		return driver.ErrBadConn
	}

	if r.currentRow >= r.maxRows || r.maxRows == 0 {
		return io.EOF
	}

	dest[0] = int64(r.currentRow + 1)                   // id
	dest[1] = fmt.Sprintf("test_name_%d", r.currentRow) // name
	dest[2] = time.Now()                                // created_at

	r.currentRow++
	return nil
}

// fakeTx is a fake transaction.
type fakeTx struct{}

func (tx *fakeTx) Commit() error {
	return nil
}

func (tx *fakeTx) Rollback() error {
	return nil
}

// QueryContext implements driver.QueryerContext
func (c *fakeConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.closed {
		return nil, driver.ErrBadConn
	}
	// simple limit
	if strings.Contains(query, "limit") && len(args) > 0 {
		if limit, ok := args[0].Value.(int64); ok && limit > 0 {
			return &fakeRows{maxRows: int(limit)}, nil
		}
	}
	return &fakeRows{maxRows: 3}, nil
}

// ExecContext implements driver.ExecerContext
func (c *fakeConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if c.closed {
		return nil, driver.ErrBadConn
	}
	return &fakeResult{}, nil
}

// BeginTx implements driver.ConnBeginTx
func (c *fakeConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if c.closed {
		return nil, driver.ErrBadConn
	}
	return &fakeTx{}, nil
}

// Ensure all interfaces are implemented
var (
	_ driver.Driver         = (*fakeDriver)(nil)
	_ driver.Conn           = (*fakeConn)(nil)
	_ driver.Stmt           = (*fakeStmt)(nil)
	_ driver.Result         = (*fakeResult)(nil)
	_ driver.Rows           = (*fakeRows)(nil)
	_ driver.Tx             = (*fakeTx)(nil)
	_ driver.QueryerContext = (*fakeConn)(nil)
	_ driver.ExecerContext  = (*fakeConn)(nil)
	_ driver.ConnBeginTx    = (*fakeConn)(nil)
)

type user struct {
	ID        int64     `column:"id"`
	Name      string    `column:"name"`
	CreatedAt time.Time `column:"created_at"`
}

func TestRegular_Query(t *testing.T) {
	db, err := sql.Open("fake", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tests := []struct {
		name    string
		query   string
		wantErr bool
		limit   int64
	}{
		{
			name:    "simple query",
			query:   "SELECT id, name, created_at FROM users",
			wantErr: false,
		},
		{
			name:    "simple query with limit",
			query:   "SELECT id, name, created_at FROM users limit ?",
			wantErr: false,
			limit:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := db.Query(tt.query, tt.limit)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()

			var users []user

			for rows.Next() {
				var user user
				if err = rows.Scan(&user.ID, &user.Name, &user.CreatedAt); err != nil {
					t.Fatal(err)
				}
				t.Log(user)
				users = append(users, user)
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			if tt.limit == 1 && len(users) != 1 {
				t.Errorf("expected 1 user, got %d", len(users))
			}
		})
	}
}
