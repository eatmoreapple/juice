package juice

import (
	"context"
	"database/sql"
	"embed"
	_ "github.com/go-sql-driver/mysql"
	"testing"
)

//go:embed testdata/configuration
var config embed.FS

func newEngine() (*Engine, error) {
	cfg, err := NewXMLConfigurationWithFS(config, "testdata/configuration/juice.xml")
	if err != nil {
		return nil, err
	}
	return Default(cfg)
}

func Hello() {}

func TestEngineConnect(t *testing.T) {
	engine, err := newEngine()
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	if err := engine.DB().Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}
}

func TestSelectHello(t *testing.T) {
	engine, err := newEngine()
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	var name string
	rows, err := engine.Object(Hello).QueryContext(context.TODO(), nil)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatalf("No rows returned")
	}
	if err := rows.Scan(&name); err != nil {
		t.Fatalf("Failed to scan: %v", err)
	}
	if name != "hello world" {
		t.Fatalf("Unexpected name: %s", name)
	}
}

type User struct {
	Id   int    `column:"id" autoincr:"true" param:"id"`
	Name string `column:"name" param:"name"`
	Age  int    `column:"age" param:"age"`
}

type UserRepository interface {
	Create(ctx context.Context, user *User) (sql.Result, error)
	GetById(ctx context.Context, id int) (*User, error)
	UpdateNameById(ctx context.Context, id int, name string) (sql.Result, error)
	DeleteById(ctx context.Context, id int) (sql.Result, error)
}

type userRepositoryImpl struct{}

func (u userRepositoryImpl) Create(ctx context.Context, user *User) (sql.Result, error) {
	manager := ManagerFromContext(ctx)
	var iface UserRepository = u
	executor := NewGenericManager[any](manager).Object(iface.Create)
	ret, err := executor.ExecContext(ctx, user)
	return ret, err
}

func (u userRepositoryImpl) GetById(ctx context.Context, id int) (*User, error) {
	manager := ManagerFromContext(ctx)
	var iface UserRepository = u
	executor := NewGenericManager[User](manager).Object(iface.GetById)
	ret, err := executor.QueryContext(ctx, H{"id": id})
	return &ret, err
}

func (u userRepositoryImpl) UpdateNameById(ctx context.Context, id int, name string) (sql.Result, error) {
	manager := ManagerFromContext(ctx)
	var iface UserRepository = u
	executor := NewGenericManager[any](manager).Object(iface.UpdateNameById)
	ret, err := executor.ExecContext(ctx, H{"id": id, "name": name})
	return ret, err
}

func (u userRepositoryImpl) DeleteById(ctx context.Context, id int) (sql.Result, error) {
	manager := ManagerFromContext(ctx)
	var iface UserRepository = u
	executor := NewGenericManager[any](manager).Object(iface.DeleteById)
	ret, err := executor.ExecContext(ctx, H{"id": id})
	return ret, err
}

func TestEngineCRUD(t *testing.T) {
	engine, err := newEngine()
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	_, err = engine.DB().Exec(`
		CREATE TABLE IF NOT EXISTS test_users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(50) NOT NULL,
			age INT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	defer func() {
		_, err := engine.DB().Exec("DROP TABLE IF EXISTS test_users")
		if err != nil {
			t.Errorf("Failed to clean up test table: %v", err)
		}
	}()

	ctx := ContextWithManager(context.Background(), engine)

	var repo UserRepository = userRepositoryImpl{}

	t.Run("Create", func(t *testing.T) {
		result, err := engine.Object(repo.Create).ExecContext(context.TODO(), User{Age: 18, Name: "eatmoreapple"})
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
		if id == 0 {
			t.Fatalf("Failed to create test user: invalid id")
		}
	})

	t.Run("GetById", func(t *testing.T) {
		user, err := repo.GetById(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to get test user: %v", err)
		}
		if user.Id != 1 {
			t.Fatalf("Failed to get test user: invalid id")
		}
	})

	t.Run("UpdateNameById", func(t *testing.T) {
		result, err := repo.UpdateNameById(ctx, 1, "apple")
		if err != nil {
			t.Fatalf("Failed to update test user: %v", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			t.Fatalf("Failed to update test user: %v", err)
		}
		if rows == 0 {
			t.Fatalf("Failed to update test user: no rows affected")
		}
		// check if the name is updated
		user, err := repo.GetById(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to get test user: %v", err)
		}
		if user.Name != "apple" {
			t.Fatalf("Failed to update test user: invalid name")
		}
	})

	t.Run("DeleteById", func(t *testing.T) {
		result, err := repo.DeleteById(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to delete test user: %v", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			t.Fatalf("Failed to delete test user: %v", err)
		}
		if rows == 0 {
			t.Fatalf("Failed to delete test user: no rows affected")
		}
		// check if the user is deleted
		user, err := repo.GetById(ctx, 1)
		if err == nil {
			t.Fatalf("Failed to delete test user: user still exists")
		}
		if user != nil {
			t.Fatalf("Failed to delete test user: user still exists")
		}
	})
}
