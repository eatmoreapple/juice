package juice

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"
)

type TestResult struct {
	Id        int       `column:"id"`
	Name      string    `column:"name"`
	Age       int       `column:"age"`
	CreatedAt time.Time `column:"created_at"`
}

// TestRowScanner implements RowScanner interface for testing
type TestRowScanner struct {
	Value string
}

func (t *TestRowScanner) ScanRows(rows *sql.Rows) error {
	return rows.Scan(&t.Value)
}

func setupResultMapTestDB(t *testing.T) *sql.DB {
	engine, err := newEngine()
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	db := engine.DB()

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS test_results (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(50) NOT NULL,
			age INT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Insert test data
	_, err = db.ExecContext(context.Background(),
		`INSERT INTO test_results (name, age) VALUES (?, ?), (?, ?)`,
		"Alice", 20,
		"Bob", 25,
	)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	return db
}

func cleanupResultMapTestDB(t *testing.T, db *sql.DB) {
	_, err := db.Exec("DROP TABLE IF EXISTS test_results")
	if err != nil {
		t.Errorf("Failed to clean up test table: %v", err)
	}
}

func TestSingleRowResultMap(t *testing.T) {
	db := setupResultMapTestDB(t)
	defer cleanupResultMapTestDB(t, db)

	t.Run("Valid Single Row", func(t *testing.T) {
		rows, err := db.QueryContext(context.Background(), "SELECT * FROM test_results WHERE id = 1")
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		defer rows.Close()

		var result TestResult
		mapper := SingleRowResultMap{}
		err = mapper.MapTo(reflect.ValueOf(&result), rows)
		if err != nil {
			t.Fatalf("Failed to map row: %v", err)
		}

		if result.Id != 1 || result.Name != "Alice" || result.Age != 20 {
			t.Errorf("Unexpected result: %+v", result)
		}
	})

	t.Run("No Rows", func(t *testing.T) {
		rows, err := db.QueryContext(context.Background(), "SELECT * FROM test_results WHERE id = 999")
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		defer rows.Close()

		var result TestResult
		mapper := SingleRowResultMap{}
		err = mapper.MapTo(reflect.ValueOf(&result), rows)
		if err != sql.ErrNoRows {
			t.Errorf("Expected sql.ErrNoRows, got %v", err)
		}
	})

	t.Run("Too Many Rows", func(t *testing.T) {
		rows, err := db.QueryContext(context.Background(), "SELECT * FROM test_results")
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		defer rows.Close()

		var result TestResult
		mapper := SingleRowResultMap{}
		err = mapper.MapTo(reflect.ValueOf(&result), rows)
		if !errors.Is(err, ErrTooManyRows) {
			t.Errorf("Expected ErrTooManyRows, got %v", err)
		}
	})

	t.Run("Non-Pointer Value", func(t *testing.T) {
		rows, err := db.QueryContext(context.Background(), "SELECT * FROM test_results LIMIT 1")
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		defer rows.Close()

		var result TestResult
		mapper := SingleRowResultMap{}
		err = mapper.MapTo(reflect.ValueOf(result), rows)
		if !errors.Is(err, ErrPointerRequired) {
			t.Errorf("Expected ErrPointerRequired, got %v", err)
		}
	})
}

func TestMultiRowsResultMap(t *testing.T) {
	db := setupResultMapTestDB(t)
	defer cleanupResultMapTestDB(t, db)

	t.Run("Multiple Rows", func(t *testing.T) {
		rows, err := db.QueryContext(context.Background(), "SELECT * FROM test_results ORDER BY id")
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		defer rows.Close()

		var results []TestResult
		mapper := MultiRowsResultMap{}
		err = mapper.MapTo(reflect.ValueOf(&results), rows)
		if err != nil {
			t.Fatalf("Failed to map rows: %v", err)
		}

		if len(results) != 2 {
			t.Fatalf("Expected 2 results, got %d", len(results))
		}

		expected := []TestResult{
			{Id: 1, Name: "Alice", Age: 20},
			{Id: 2, Name: "Bob", Age: 25},
		}

		for i, exp := range expected {
			if results[i].Id != exp.Id || results[i].Name != exp.Name || results[i].Age != exp.Age {
				t.Errorf("Row %d mismatch: expected %+v, got %+v", i, exp, results[i])
			}
		}
	})

	t.Run("Empty Result Set", func(t *testing.T) {
		rows, err := db.QueryContext(context.Background(), "SELECT * FROM test_results WHERE id > 999")
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		defer rows.Close()

		var results []TestResult
		mapper := MultiRowsResultMap{}
		err = mapper.MapTo(reflect.ValueOf(&results), rows)
		if err != nil {
			t.Fatalf("Failed to map rows: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("Expected empty result set, got %d results", len(results))
		}
	})

	t.Run("Custom New Function", func(t *testing.T) {
		rows, err := db.QueryContext(context.Background(), "SELECT * FROM test_results ORDER BY id")
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		defer rows.Close()

		var results []*TestResult
		mapper := MultiRowsResultMap{
			New: func() reflect.Value {
				return reflect.ValueOf(&TestResult{})
			},
		}
		err = mapper.MapTo(reflect.ValueOf(&results), rows)
		if err != nil {
			t.Fatalf("Failed to map rows: %v", err)
		}

		if len(results) != 2 {
			t.Fatalf("Expected 2 results, got %d", len(results))
		}

		expected := []*TestResult{
			{Id: 1, Name: "Alice", Age: 20},
			{Id: 2, Name: "Bob", Age: 25},
		}

		for i, exp := range expected {
			if results[i].Id != exp.Id || results[i].Name != exp.Name || results[i].Age != exp.Age {
				t.Errorf("Row %d mismatch: expected %+v, got %+v", i, exp, results[i])
			}
		}
	})

	t.Run("RowScanner Implementation", func(t *testing.T) {
		rows, err := db.QueryContext(context.Background(), "SELECT name FROM test_results ORDER BY id")
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		defer rows.Close()

		var results []*TestRowScanner
		mapper := MultiRowsResultMap{}
		err = mapper.MapTo(reflect.ValueOf(&results), rows)
		if err != nil {
			t.Fatalf("Failed to map rows: %v", err)
		}

		expected := []string{"Alice", "Bob"}
		if len(results) != len(expected) {
			t.Fatalf("Expected %d results, got %d", len(expected), len(results))
		}

		for i, exp := range expected {
			if results[i].Value != exp {
				t.Errorf("Row %d mismatch: expected %s, got %s", i, exp, results[i].Value)
			}
		}
	})
}

func TestRowDestination(t *testing.T) {
	db := setupResultMapTestDB(t)
	defer cleanupResultMapTestDB(t, db)

	t.Run("Struct Field Mapping", func(t *testing.T) {
		rows, err := db.QueryContext(context.Background(), "SELECT * FROM test_results WHERE id = 1")
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			t.Fatalf("Failed to get columns: %v", err)
		}

		var result TestResult
		dest := &rowDestination{}
		destinations, err := dest.Destination(reflect.ValueOf(&result).Elem(), columns)
		if err != nil {
			t.Fatalf("Failed to create destinations: %v", err)
		}

		if len(destinations) != len(columns) {
			t.Errorf("Expected %d destinations, got %d", len(columns), len(destinations))
		}

		if !rows.Next() {
			t.Fatal("Expected at least one row")
		}

		err = rows.Scan(destinations...)
		if err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}

		if result.Id != 1 || result.Name != "Alice" || result.Age != 20 {
			t.Errorf("Unexpected result: %+v", result)
		}
	})
}
