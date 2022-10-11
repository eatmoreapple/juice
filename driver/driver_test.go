package driver

import (
	"strconv"
	"testing"
)

func TestMySQLDriver(t *testing.T) {
	driver := MySQLDriver{}
	translator := driver.Translate()
	if translator.Translate("foo") != "?" {
		t.Fatal("failed to translate")
	}
}

func TestSQLiteDriver(t *testing.T) {
	driver := SQLiteDriver{}
	translator := driver.Translate()
	if translator.Translate("foo") != "?" {
		t.Fatal("failed to translate")
	}
}

func TestPostgresDriver(t *testing.T) {
	driver := PostgresDriver{}
	translator := driver.Translate()
	for i := 0; i < 10; i++ {
		if translator.Translate("foo") != "$"+strconv.Itoa(i+1) {
			t.Fatal("failed to translate")
		}
	}
	translator = driver.Translate()
	for i := 0; i < 10; i++ {
		if translator.Translate("bar") != "$"+strconv.Itoa(i+1) {
			t.Fatal("failed to translate")
		}
	}
}

func TestOracleDriver(t *testing.T) {
	driver := OracleDriver{}
	translator := driver.Translate()
	for i := 0; i < 10; i++ {
		if translator.Translate("foo") != ":"+strconv.Itoa(i+1) {
			t.Fatal("failed to translate")
		}
	}
	translator = driver.Translate()
	for i := 0; i < 10; i++ {
		if translator.Translate("bar") != ":"+strconv.Itoa(i+1) {
			t.Fatal("failed to translate")
		}
	}
}
