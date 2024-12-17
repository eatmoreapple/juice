package juice

import (
	"database/sql"
	"testing"
)

func TestBind(t *testing.T) {
	db, err := sql.Open("fake", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "simple query",
			query:   "SELECT id, name, created_at FROM users",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := db.Query(tt.query)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()

			users, err := Bind[[]user](rows)

			if err != nil {
				t.Fatal(err)
			}
			for _, user := range users {
				t.Log(user)
			}
		})
		t.Run(tt.name, func(t *testing.T) {
			rows, err := db.Query(tt.query)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()

			users, err := Bind[[]*user](rows)

			if err != nil {
				t.Fatal(err)
			}
			for _, user := range users {
				t.Log(user)
			}
		})
	}
}

func TestList(t *testing.T) {
	db, err := sql.Open("fake", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "simple query",
			query:   "SELECT id, name, created_at FROM users",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := db.Query(tt.query)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()

			users, err := List[user](rows)

			if err != nil {
				t.Fatal(err)
			}
			for _, user := range users {
				t.Log(user)
			}
		})
		t.Run(tt.name, func(t *testing.T) {
			rows, err := db.Query(tt.query)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()

			users, err := List[*user](rows)

			if err != nil {
				t.Fatal(err)
			}
			for _, user := range users {
				t.Log(user)
			}
		})
	}
}
