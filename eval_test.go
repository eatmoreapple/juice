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

func TestLen(t *testing.T) {
	param := Param{
		"a": reflect.ValueOf([]interface{}{"a", "b", "c"}),
		"b": reflect.ValueOf("aaa"),
		"c": reflect.ValueOf(map[string]interface{}{"a": "a", "b": "b", "c": "c"}),
	}
	result, err := Eval(`len(a)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Int() != 3 {
		t.Error("eval error")
		return
	}
	result, err = Eval(`len(b)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Int() != 3 {
		t.Error("eval error")
		return
	}
	result, err = Eval(`len(c)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Int() != 3 {
		t.Error("eval error")
		return
	}
}

func TestSubStr(t *testing.T) {
	param := Param{
		"a": reflect.ValueOf("eatmoreapple"),
	}
	result, err := Eval(`substr(a, 0, 3)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "eat" {
		t.Error("eval error")
		return
	}
	result, err = Eval(`substr(a, 3, 4)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "more" {
		t.Error("eval error")
		return
	}
	result, err = Eval(`substr(a, 7, 5)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "apple" {
		t.Error("eval error")
		return
	}
}

func TestSubJoin(t *testing.T) {
	param := Param{
		"a": reflect.ValueOf([]string{"eat", "more", "apple"}),
	}
	result, err := Eval(`join(a, "")`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "eatmoreapple" {
		t.Error("eval error")
		return
	}
}

func TestContains(t *testing.T) {
	param := Param{
		"a": reflect.ValueOf([]string{"eat", "more", "apple"}),
		"b": reflect.ValueOf([]int64{1, 2, 3}),
	}
	result, err := Eval(`contains(a, "eat")`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}

	result, err = Eval(`contains("eatmoreapple", "eat")`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}

	result, err = Eval(`contains(b, 3)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}

	result, err = Eval(`contains(b, 4)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}
}

func TestSlice(t *testing.T) {
	param := Param{
		"a": reflect.ValueOf([]string{"eat", "more", "apple"}),
	}
	result, err := Eval(`slice(a, 0, 1)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Len() != 1 {
		t.Error("eval error")
		return
	}
	if result.Index(0).Interface() != "eat" {
		t.Error("eval error")
		return
	}
}

func TestLparenRparen(t *testing.T) {
	result, err := Eval(`2 * (2 + 5) == 14`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}
	result, err = Eval(`2 * (2 + 5) / 2`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Int() != 7 {
		t.Error("eval error")
		return
	}
}

func TestComment(t *testing.T) {
	result, err := Eval(`2 * (2 + 5) + 1 // 2 * (2 + 5) == 14`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Int() != 15 {
		t.Error("eval error")
		return
	}
}

func TestUnaryExpr(t *testing.T) {
	result, err := Eval(`-2`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Int() != -2 {
		t.Error("eval error")
		return
	}
}

func TestUnaryExpr2(t *testing.T) {
	result, err := Eval(`-2 * 3`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Int() != -6 {
		t.Error("eval error")
		return
	}
}

func TestIndexExprSlice(t *testing.T) {
	param := Param{
		"a": reflect.ValueOf([]string{"eat", "more", "apple"}),
	}
	result, err := Eval(`a[0]`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "eat" {
		t.Error("eval error")
		return
	}
	result, err = Eval(`a[0] + a[1]`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "eatmore" {
		t.Error("eval error")
		return
	}
}

func TestIndexExprMap(t *testing.T) {
	param := Param{
		"a": reflect.ValueOf(map[string]string{
			"eat": "more",
		}),
	}
	result, err := Eval(`a["eat"]`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "more" {
		t.Error("eval error")
		return
	}
}

func TestStarExpr(t *testing.T) {
	result, err := Eval(`*2`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Int() != 2 {
		t.Error("eval error")
		return
	}
	result, err = Eval(`2 *2`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Int() != 4 {
		t.Error("eval error")
		return
	}
}

func TestSliceExpr(t *testing.T) {
	param := Param{
		"a": reflect.ValueOf([]string{"eat", "more", "apple"}),
	}
	result, err := Eval(`a[:]`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Len() != 3 {
		t.Errorf("eval error: %d", result.Len())
		return
	}
	if result.Index(0).Interface() != "eat" {
		t.Error("eval error")
		return
	}
	if result.Index(1).Interface() != "more" {
		t.Error("eval error")
		return
	}
	if result.Index(2).Interface() != "apple" {
		t.Error("eval error")
		return
	}
	result, err = Eval(`a[1:]`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Len() != 2 {
		t.Errorf("eval error: %d", result.Len())
		return
	}
	if result.Index(0).Interface() != "more" {
		t.Error("eval error")
		return
	}
	if result.Index(1).Interface() != "apple" {
		t.Error("eval error")
		return
	}
	result, err = Eval(`a[1:2]`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Len() != 1 {
		t.Errorf("eval error: %d", result.Len())
		return
	}
	if result.Index(0).Interface() != "more" {
		t.Error("eval error")
		return
	}
}
