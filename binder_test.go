package juice

import (
	"context"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"testing"
)

type TestUser struct {
	Id   int    `column:"id" autoincr:"true"`
	Name string `column:"name"`
	Age  int    `column:"age"`
}

func setupTestDB(t *testing.T) *sql.DB {
	engine, err := newEngine()
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	db := engine.DB()

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS test_users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(50) NOT NULL,
			age INT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`INSERT INTO test_users (name, age) VALUES (?, ?)`, "Alice", 20)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	return db
}

func cleanupTestDB(t *testing.T, db *sql.DB) {
	_, err := db.Exec("DROP TABLE IF EXISTS test_users")
	if err != nil {
		t.Errorf("Failed to clean up test table: %v", err)
	}
}

func TestBindSingleRow(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	rows, err := db.QueryContext(context.Background(), "SELECT * FROM test_users WHERE id = 1")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	defer rows.Close()

	user, err := Bind[TestUser](rows)
	if err != nil {
		t.Fatalf("Failed to bind: %v", err)
	}

	if user.Id != 1 {
		t.Errorf("Expected id 1, got %d", user.Id)
	}
	if user.Name != "Alice" {
		t.Errorf("Expected name 'Alice', got %s", user.Name)
	}
	if user.Age != 20 {
		t.Errorf("Expected age 20, got %d", user.Age)
	}
}

func TestBindMultipleRows(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Insert another row
	_, err := db.Exec(`INSERT INTO test_users (name, age) VALUES (?, ?)`, "Bob", 25)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	rows, err := db.QueryContext(context.Background(), "SELECT * FROM test_users")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	defer rows.Close()

	users, err := Bind[[]TestUser](rows)
	if err != nil {
		t.Fatalf("Failed to bind: %v", err)
	}

	if len(users) != 2 {
		t.Fatalf("Expected 2 users, got %d", len(users))
	}

	// Check first user
	if users[0].Name != "Alice" || users[0].Age != 20 {
		t.Errorf("First user mismatch: got %+v", users[0])
	}

	// Check second user
	if users[1].Name != "Bob" || users[1].Age != 25 {
		t.Errorf("Second user mismatch: got %+v", users[1])
	}
}

func TestList(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Insert another row
	_, err := db.Exec(`INSERT INTO test_users (name, age) VALUES (?, ?)`, "Bob", 25)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	rows, err := db.QueryContext(context.Background(), "SELECT * FROM test_users")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	defer rows.Close()

	users, err := List[TestUser](rows)
	if err != nil {
		t.Fatalf("Failed to list: %v", err)
	}

	if len(users) != 2 {
		t.Fatalf("Expected 2 users, got %d", len(users))
	}

	// Verify data
	expectedUsers := []TestUser{
		{Id: 1, Name: "Alice", Age: 20},
		{Id: 2, Name: "Bob", Age: 25},
	}

	for i, expected := range expectedUsers {
		if users[i].Id != expected.Id ||
			users[i].Name != expected.Name ||
			users[i].Age != expected.Age {
			t.Errorf("User %d mismatch: expected %+v, got %+v", i, expected, users[i])
		}
	}
}

func TestBindWithResultMap(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Test SingleRowResultMap
	t.Run("SingleRow", func(t *testing.T) {
		rows, err := db.QueryContext(context.Background(), "SELECT * FROM test_users WHERE id = 1")
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		defer rows.Close()

		user, err := BindWithResultMap[TestUser](rows, SingleRowResultMap{})
		if err != nil {
			t.Fatalf("Failed to bind with SingleRowResultMap: %v", err)
		}

		if user.Id != 1 || user.Name != "Alice" || user.Age != 20 {
			t.Errorf("User mismatch: got %+v", user)
		}
	})

	// Test MultiRowsResultMap
	t.Run("MultiRows", func(t *testing.T) {
		// Insert another row
		_, err := db.Exec(`INSERT INTO test_users (name, age) VALUES (?, ?)`, "Bob", 25)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}

		rows, err := db.QueryContext(context.Background(), "SELECT * FROM test_users")
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		defer rows.Close()

		users, err := BindWithResultMap[[]TestUser](rows, MultiRowsResultMap{})
		if err != nil {
			t.Fatalf("Failed to bind with MultiRowsResultMap: %v", err)
		}

		if len(users) != 2 {
			t.Fatalf("Expected 2 users, got %d", len(users))
		}

		expectedUsers := []TestUser{
			{Id: 1, Name: "Alice", Age: 20},
			{Id: 2, Name: "Bob", Age: 25},
		}

		for i, expected := range expectedUsers {
			if users[i].Id != expected.Id ||
				users[i].Name != expected.Name ||
				users[i].Age != expected.Age {
				t.Errorf("User %d mismatch: expected %+v, got %+v", i, expected, users[i])
			}
		}
	})
}
