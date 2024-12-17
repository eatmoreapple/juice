package juice

import (
	"database/sql"
	"reflect"
	"testing"
	"time"
)

type user2 struct {
	ID        int64     `column:"id"`
	Name      string    `column:"name"`
	CreatedAt time.Time `column:"created_at"`
}

func (u *user2) ScanRows(rows *sql.Rows) error {
	return rows.Scan(&u.ID, &u.Name, &u.CreatedAt)
}

func TestSingleRowResultMap_MapTo(t *testing.T) {
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
			name:    "single row query",
			query:   "SELECT id, name, created_at FROM users limit ?",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := db.Query(tt.query, 1)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()

			var user user2
			mapper := SingleRowResultMap{}
			err = mapper.MapTo(reflect.ValueOf(&user), rows)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapTo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Log(user)
		})
	}
}

func TestMultiRowsResultMap_MapTo(t *testing.T) {
	db, err := sql.Open("fake", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tests := []struct {
		name      string
		query     string
		wantErr   bool
		wantEmpty bool
	}{
		{
			name:      "multiple rows",
			query:     "SELECT id, name, created_at FROM users",
			wantErr:   false,
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" with slice", func(t *testing.T) {
			rows, err := db.Query(tt.query)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()

			var users []user2
			mapper := MultiRowsResultMap{}
			err = mapper.MapTo(reflect.ValueOf(&users), rows)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapTo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.wantEmpty && len(users) > 0 {
					t.Error("Expected empty result")
				}
				if !tt.wantEmpty && len(users) == 0 {
					t.Error("Expected non-empty result")
				}
			}
		})
		t.Run(tt.name+" with slice with new func", func(t *testing.T) {
			rows, err := db.Query(tt.query)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()

			var users []user2
			mapper := MultiRowsResultMap{
				New: func() reflect.Value {
					return reflect.ValueOf(&user2{})
				},
			}
			err = mapper.MapTo(reflect.ValueOf(&users), rows)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapTo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.wantEmpty && len(users) > 0 {
					t.Error("Expected empty result")
				}
				if !tt.wantEmpty && len(users) == 0 {
					t.Error("Expected non-empty result")
				}
			}
		})

		t.Run(tt.name+" with pointer slice", func(t *testing.T) {
			rows, err := db.Query(tt.query)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()

			var users []*user2
			mapper := MultiRowsResultMap{}
			err = mapper.MapTo(reflect.ValueOf(&users), rows)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapTo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.wantEmpty && len(users) > 0 {
					t.Error("Expected empty result")
				}
				if !tt.wantEmpty && len(users) == 0 {
					t.Error("Expected non-empty result")
				}
			}
		})

		t.Run(tt.name+" with pointer slice with new func", func(t *testing.T) {
			rows, err := db.Query(tt.query)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()

			var users []*user2
			mapper := MultiRowsResultMap{
				New: func() reflect.Value {
					return reflect.ValueOf(&user2{})
				},
			}
			err = mapper.MapTo(reflect.ValueOf(&users), rows)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapTo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.wantEmpty && len(users) > 0 {
					t.Error("Expected empty result")
				}
				if !tt.wantEmpty && len(users) == 0 {
					t.Error("Expected non-empty result")
				}
			}
		})
	}
}
