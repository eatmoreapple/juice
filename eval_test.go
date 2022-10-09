package juice

import (
	"go/parser"
	"reflect"
	"testing"
)

func TestEval(t *testing.T) {
	param := Param{
		"id":   reflect.ValueOf(1),
		"age":  reflect.ValueOf(18),
		"name": reflect.ValueOf("eatmoreapple"),
	}
	result, err := Eval(`id > 0 && id < 2`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}

	result, err = Eval(`age == 17 + 1 && age == 36 / 2 && age == 9 * 2 && age == 19 -1`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}

	result, err = Eval(`name == "eatmoreapple"`, param)
	if err != nil {
		t.Error(err)
		return
	}

	if !result.Bool() {
		t.Error("eval error")
		return
	}

	result, err = Eval(`"eat" + "more" + "apple"`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "eatmoreapple" {
		t.Error("eval error")
		return
	}
}

func BenchmarkEval(b *testing.B) {
	param := Param{
		"id":   reflect.ValueOf(1),
		"age":  reflect.ValueOf(18),
		"name": reflect.ValueOf("eatmoreapple"),
	}
	for i := 0; i < b.N; i++ {
		value, err := Eval(`id > 0 && id < 2 && name == "eatmoreapple"`, param)
		if err != nil {
			b.Error(err)
			return
		}
		if !value.Bool() {
			b.Error("eval error")
			return
		}
	}
	// BenchmarkEval-8   	 1047154	      1111 ns/op
}

func BenchmarkEval2(b *testing.B) {
	param := Param{
		"id":   reflect.ValueOf(1),
		"age":  reflect.ValueOf(18),
		"name": reflect.ValueOf("eatmoreapple"),
	}
	expr, err := parser.ParseExpr(`id > 0 && id < 2 && name == "eatmoreapple"`)
	if err != nil {
		b.Error(err)
		return
	}
	for i := 0; i < b.N; i++ {
		value, err := eval(expr, param)
		if err != nil {
			b.Error(err)
			return
		}
		if !value.Bool() {
			b.Error("eval error")
			return
		}
	}
	// BenchmarkEval2-8   	 5736370	       180.8 ns/op
}
