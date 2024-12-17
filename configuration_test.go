package juice

import (
	"embed"
	"testing"
)

//go:embed testdata/configuration
var cfg embed.FS

func TestNewXMLConfigurationWithFS(t *testing.T) {
	_, err := NewXMLConfigurationWithFS(cfg, "testdata/configuration/juice.xml")
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewXMLConfiguration(t *testing.T) {
	_, err := NewXMLConfiguration("testdata/configuration/juice.xml")
	if err != nil {
		t.Fatal(err)
	}
}
