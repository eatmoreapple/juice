package juice

import (
	"embed"
	_ "github.com/go-sql-driver/mysql"
	"testing"
)

//go:embed testdata/configuration
var config embed.FS

func TestEngineConnect(t *testing.T) {
	cfg, err := NewXMLConfigurationWithFS(config, "testdata/configuration/juice.xml")
	if err != nil {
		t.Fatalf("Failed to create new configuration: %v", err)
	}
	engine, err := Default(cfg)
	if err != nil {
		t.Fatalf("Failed to create new engine: %v", err)
	}
	if err := engine.DB().Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}
}
