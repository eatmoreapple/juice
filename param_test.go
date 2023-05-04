package juice

import (
	"reflect"
	"testing"
)

func TestParam_Get(t *testing.T) {
	param := Param{
		"list": reflect.ValueOf([]any{1, 2, 3}),
	}
	value, exists := param.Get("list.1")
	if !exists {
		t.Error("exists error")
		return
	}
	if value.Int() != 2 {
		t.Error("value error")
		return
	}
}
