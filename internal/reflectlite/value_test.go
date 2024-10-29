package reflectlite

import (
	"reflect"
	"testing"
)

func TestValue_FindFieldFromTag(t *testing.T) {
	type A struct {
		AName string `param:"a_name"`
	}

	type B struct {
		BName string `param:"b_name"`
		A
	}

	var b B

	b.AName = "a_name"
	b.BName = "b_name"

	value := ValueFrom(reflect.ValueOf(b))

	v := value.FindFieldFromTag("param", "a_name")
	if !v.IsValid() {
		t.Error("expect a_name")
	}
	if v.String() != "a_name" {
		t.Error("expect a_name")
	}
}

func TestValue_GetFieldIndexesFromTag(t *testing.T) {
	type A struct {
		AName string `param:"a_name"`
	}

	type B struct {
		BName string `param:"b_name"`
		A
	}

	var b B

	b.AName = "a_name"
	b.BName = "b_name"

	value := ValueFrom(reflect.ValueOf(b))

	// Test finding field index by tag
	indexes, ok := value.GetFieldIndexesFromTag("param", "a_name")
	if !ok {
		t.Error("expected to find a_name")
	}
	if len(indexes) != 2 || indexes[0] != 1 || indexes[1] != 0 {
		t.Errorf("expected indexes [1 0], got %v", indexes)
	}
	t.Log(indexes)

	// Test not finding field index by non-existent tag
	indexes, ok = value.GetFieldIndexesFromTag("param", "non_existent")
	if ok {
		t.Error("expected not to find non_existent")
	}
	if indexes != nil {
		t.Errorf("expected nil indexes, got %v", indexes)
	}

	// Test not finding field index in non-struct type
	nonStructValue := ValueFrom(reflect.ValueOf("string"))
	indexes, ok = nonStructValue.GetFieldIndexesFromTag("param", "a_name")
	if ok {
		t.Error("expected not to find a_name in non-struct type")
	}
	if indexes != nil {
		t.Errorf("expected nil indexes, got %v", indexes)
	}
}
