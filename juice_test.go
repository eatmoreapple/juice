package juice

import (
	"context"
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
