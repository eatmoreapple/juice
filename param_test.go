package juice

import (
	"reflect"
	"testing"
)

func TestParam_Get(t *testing.T) {
	param := H{
		"list": []any{1, 2, nil},
	}.AsParam()
	value, exists := param.Get("list.1")
	if !exists {
		t.Error("exists error")
		return
	}
	compare := new(int)
	*compare = 2
	result, err := eql(value, reflect.ValueOf(compare))
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("value error")
		return
	}

	value, _ = param.Get("list.2")
	result, err = eql(value, reflect.ValueOf(nil))
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("value error")
		return
	}
}
