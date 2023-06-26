package driver

import (
	"strconv"
	"testing"
)

func TestMySQLDriver(t *testing.T) {
	driver := MySQLDriver{}
	translator := driver.Translator()
	if translator.Translate("foo") != "?" {
		t.Fatal("failed to translate")
	}
}

func TestSQLiteDriver(t *testing.T) {
	driver := SQLiteDriver{}
	translator := driver.Translator()
	if translator.Translate("foo") != "?" {
		t.Fatal("failed to translate")
	}
}

func TestPostgresDriver(t *testing.T) {
	driver := PostgresDriver{}
	translator := driver.Translator()
	for i := 0; i < 10; i++ {
		if translator.Translate("foo") != "$"+strconv.Itoa(i+1) {
			t.Fatal("failed to translate")
		}
	}
	translator = driver.Translator()
	for i := 0; i < 10; i++ {
		if translator.Translate("bar") != "$"+strconv.Itoa(i+1) {
			t.Fatal("failed to translate")
		}
	}
}

func TestOracleDriver(t *testing.T) {
	driver := OracleDriver{}
	translator := driver.Translator()
	for i := 0; i < 10; i++ {
		if translator.Translate("foo") != ":"+strconv.Itoa(i+1) {
			t.Fatal("failed to translate")
		}
	}
	translator = driver.Translator()
	for i := 0; i < 10; i++ {
		if translator.Translate("bar") != ":"+strconv.Itoa(i+1) {
			t.Fatal("failed to translate")
		}
	}
}
