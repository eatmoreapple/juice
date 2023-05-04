package juice

import (
	"testing"
)

func TestParam_Get(t *testing.T) {
	param := H{
		"list": []any{1, 2, 3},
	}.AsParam()
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
