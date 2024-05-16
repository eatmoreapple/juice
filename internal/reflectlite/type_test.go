package reflectlite

import (
	"testing"
)

func TestTypeIdentify_BasicType(t *testing.T) {
	result := TypeIdentify[int]()
	if result != "int" {
		t.Errorf("Expected 'int', got '%s'", result)
	}
}

func TestTypeIdentify_StructType(t *testing.T) {
	type testType struct {
		field string
	}
	{
		// for lint no unused variable
		var testTypeInstance testType
		testTypeInstance.field = "test"
	}
	result := TypeIdentify[testType]()
	if result != "github.com/eatmoreapple/juice/internal/reflectlite.testType" {
		t.Errorf("Expected 'reflectlite.testType', got '%s'", result)
	}
}

func TestTypeIdentify_SliceType(t *testing.T) {
	type testType []int
	result := TypeIdentify[testType]()
	if result != "slice[int]" {
		t.Errorf("Expected 'slice[int]', got '%s'", result)
	}
}

func TestTypeIdentify_MapType(t *testing.T) {
	type testType map[string]int
	result := TypeIdentify[testType]()
	if result != "map[string]int" {
		t.Errorf("Expected 'map[string]int', got '%s'", result)
	}
}

func TestTypeIdentify_PointerType(t *testing.T) {
	type testType *int
	result := TypeIdentify[testType]()
	if result != "ptr[int]" {
		t.Errorf("Expected 'ptr[int]', got '%s'", result)
	}
}

func TestTypeIdentify_AnonymousStruct(t *testing.T) {
	type testType struct {
		field string // nolint:unused
	}
	result := TypeIdentify[struct{ testType }]()
	if result != "struct { reflectlite.testType }" {
		t.Errorf("Expected 'struct { reflectlite.testType }', got '%s'", result)
	}
}
