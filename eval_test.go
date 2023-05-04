package juice

import (
	"go/parser"
	"reflect"
	"testing"
)

func testEval(expr string, v any) (result reflect.Value, err error) {
	param := newGenericParam(v, "")
	return Eval(expr, param)
}

func TestEval(t *testing.T) {
	param := H{
		"id":   1,
		"age":  18,
		"name": "eatmoreapple",
	}
	result, err := testEval(`id > 0 && id < 2`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}

	result, err = testEval(`age == 17 + 1 && age == 36 / 2 && age == 9 * 2 && age == 19 -1`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}

	result, err = testEval(`name == "eatmoreapple"`, param)
	if err != nil {
		t.Error(err)
		return
	}

	if !result.Bool() {
		t.Error("eval error")
		return
	}

	result, err = testEval(`"eat" + "more" + "apple"`, nil)
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
	param := H{
		"id":   reflect.ValueOf(1),
		"age":  reflect.ValueOf(18),
		"name": reflect.ValueOf("eatmoreapple"),
	}
	for i := 0; i < b.N; i++ {
		value, err := testEval(`id > 0 && id < 2 && name == "eatmoreapple"`, param)
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
	param := H{
		"id":   1,
		"age":  18,
		"name": "eatmoreapple",
	}
	expr, err := parser.ParseExpr(`id > 0 && id < 2 && name == "eatmoreapple"`)
	if err != nil {
		b.Error(err)
		return
	}
	p := newGenericParam(param, "")
	for i := 0; i < b.N; i++ {
		value, err := eval(expr, p)
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
	param := H{
		"a": []any{"a", "b", "c"},
		"b": "aaa",
		"c": map[string]any{"a": "a", "b": "b", "c": "c"},
	}
	result, err := testEval(`len(a)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Int() != 3 {
		t.Error("eval error")
		return
	}
	result, err = testEval(`len(b)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Int() != 3 {
		t.Error("eval error")
		return
	}
	result, err = testEval(`len(c)`, param)
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
	param := H{
		"a": "eatmoreapple",
	}
	result, err := testEval(`substr(a, 0, 3)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "eat" {
		t.Error("eval error")
		return
	}
	result, err = testEval(`substr(a, 3, 4)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "more" {
		t.Error("eval error")
		return
	}
	result, err = testEval(`substr(a, 7, 5)`, param)
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
	param := H{
		"a": []string{"eat", "more", "apple"},
	}
	result, err := testEval(`join(a, "")`, param)
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
	param := H{
		"a": []string{"eat", "more", "apple"},
		"b": []int64{1, 2, 3},
	}
	result, err := testEval(`contains(a, "eat")`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}

	result, err = testEval(`contains("eatmoreapple", "eat")`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}

	result, err = testEval(`contains(b, 3)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}

	result, err = testEval(`contains(b, 4)`, param)
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
	param := H{
		"a": []string{"eat", "more", "apple"},
	}
	result, err := testEval(`slice(a, 0, 1)`, param)
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
	result, err := testEval(`2 * (2 + 5) == 14`, nil)
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
	param := H{
		"a": []string{"eat", "more", "apple"},
	}
	result, err := testEval(`a[0]`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "eat" {
		t.Error("eval error")
		return
	}
	result, err = testEval(`a[0] + a[1]`, param)
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
	param := H{
		"a": map[string]string{
			"eat": "more",
		},
	}
	result, err := testEval(`a["eat"]`, param)
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
	param := H{
		"a": []string{"eat", "more", "apple"},
	}
	result, err := testEval(`a[:]`, param)
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
	result, err = testEval(`a[1:]`, param)
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
	result, err = testEval(`a[1:2]`, param)
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

func TestAnd(t *testing.T) {
	result, err := Eval(`1 + 1 < 0 && 1 + 1 == 2`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}
	result, err = Eval(`(1 + 1 < 0) & (1 + 1 == 2)`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}
	result, err = Eval("true & false", nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}
}

func TestOr(t *testing.T) {
	result, err := Eval(`1 + 1 < 0 || 1 + 1 == 2`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}
	result, err = Eval("true | false", nil)
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}
}

func TestAndOr(t *testing.T) {
	result, err := Eval(`1 + 1 < 0 || 1 + 1 == 2 && 1 + 1 == 3`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}
}

func TestAndOr2(t *testing.T) {
	result, err := Eval(`1 + 1 < 0 && 1 + 1 == 2 || 1 + 1 == 3`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}
}

func TestNot(t *testing.T) {
	result, err := Eval(`!(1 + 1 == 2)`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}
}

func TestNot2(t *testing.T) {
	result, err := Eval(`!true`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}
}
