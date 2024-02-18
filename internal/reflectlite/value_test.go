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

	value := From(reflect.ValueOf(b))

	v := value.FindFieldFromTag("param", "a_name")
	if !v.IsValid() {
		t.Error("expect a_name")
	}
	if v.String() != "a_name" {
		t.Error("expect a_name")
	}
}
