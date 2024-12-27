package eval

import (
	"go/parser"
	"reflect"
	"testing"
)

func testEval(expr string, v any) (result reflect.Value, err error) {
	param := NewGenericParam(v, "")
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
		"id":   1,
		"age":  18,
		"name": "eatmoreapple",
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
	p := NewGenericParam(param, "")
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

func TestSlice3(t *testing.T) {
	param := H{
		"a": []string{"eat", "more", "apple"},
	}
	result, err := testEval(`a[1:2:3]`, param)
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

func TestNil(t *testing.T) {
	result, err := Eval(`nil`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.IsValid() {
		t.Error("eval error")
		return
	}
}

func TestExprNilEQ(t *testing.T) {
	result, err := Eval("a == nil", H{"a": nil}.AsParam())
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}
	var a *int
	result, err = Eval("a == nil", H{"a": a}.AsParam())
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}

	var entity struct {
		A *int `param:"a"`
	}
	result, err = Eval("a == nil", NewGenericParam(entity, ""))
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}
}

func TestExprNilNEQ(t *testing.T) {
	result, err := Eval("a != nil", H{"a": nil}.AsParam())
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}
	var a *int
	result, err = Eval("a != nil", H{"a": a}.AsParam())
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}

	var entity struct {
		A *int `param:"a"`
	}
	result, err = Eval("a != nil", NewGenericParam(entity, ""))
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}

	var a2 = new(int)
	result, err = Eval("a != nil", H{"a": a2}.AsParam())
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}

	var a3 = 1
	_, err = Eval("a != nil", H{"a": a3}.AsParam())
	if err == nil {
		t.Error(err)
		return
	} else {
		t.Log(err)
	}
}

func TestSelector(t *testing.T) {
	var entity struct {
		A int `param:"a"`
	}
	entity.A = 1
	result, err := Eval("entity.A > 0", H{"entity": entity}.AsParam())
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}
}

type testStruct struct{}

func (t testStruct) Test() (bool, error) {
	return true, nil
}

func TestSelectorFunc(t *testing.T) {
	var entity struct {
		A *testStruct `param:"a"`
	}
	entity.A = &testStruct{}
	result, err := Eval("entity.A.Test()", H{"entity": entity}.AsParam())
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}

	f := func() (string, error) {
		return "test", nil
	}

	result, err = Eval("f()", H{"f": f}.AsParam())
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "test" {
		t.Error("eval error")
		return
	}
}

func TestMapDefaultMap(t *testing.T) {
	result, err := Eval("a.b", H{"a": H{"b": 1}}.AsParam())
	if err != nil {
		t.Error(err)
		return
	}
	if result.Interface() != 1 {
		t.Error("eval error")
		return
	}

	result, err = Eval(`a["c"]`, H{"a": map[string]int{}}.AsParam())
	if err != nil {
		t.Error(err)
		return
	}
	if result.Interface() != 0 {
		t.Error("eval error")
		return
	}

	result, err = Eval(`a["c"]`, H{"a": map[string]string{}}.AsParam())
	if err != nil {
		t.Error(err)
		return
	}
	if result.Interface() != "" {
		t.Error("eval error")
		return
	}

	result, err = Eval(`a["c"]`, H{"a": map[string]float64{}}.AsParam())
	if err != nil {
		t.Error(err)
		return
	}
	if result.Interface() != 0.0 {
		t.Error("eval error")
		return
	}
}

// BenchmarkStaticExpr tests the performance of static expression evaluation
func BenchmarkStaticExpr(b *testing.B) {
	tests := []struct {
		name string
		expr string
	}{
		{"simple_bool", "1 == 1"},
		{"simple_math", "1 + 2 * 3"},
		{"complex_math", "10 + 20 * 3"},
		{"string_concat", `"hello" + "world"`},
		{"mixed_ops", "1 + 2 * 3 == 7"},
	}

	b.Run("without_optimization", func(b *testing.B) {
		for _, tt := range tests {
			b.Run(tt.name, func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := Eval(tt.expr, nil)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})

	b.Run("with_optimization", func(b *testing.B) {
		compiler := &goExprCompiler{pretreatment: exprPretreatmentChain}
		for _, tt := range tests {
			b.Run(tt.name, func(b *testing.B) {
				// Pre-compile the expression
				expr, err := compiler.Compile(tt.expr)
				if err != nil {
					b.Fatal(err)
				}
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := expr.Execute(nil)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})
}

// BenchmarkStaticExprOptimizer benchmarks the static expression optimizer
func BenchmarkStaticExprOptimizer(b *testing.B) {
	benchmarks := []struct {
		name string
		expr string
		want interface{}
	}{
		{"simple_bool", "1 == 1", true},
		{"simple_math", "1 + 2 * 3", int64(7)},
		{"complex_math", "10 + 20 * 3", int64(70)},
		{"string_concat", `"hello" + "world"`, "helloworld"},
		{"mixed_ops", "1 + 2 * 3 == 7", true},
		{"bool_chain", "true && false || true", true},
		{"math_chain", "1 + 2 + 3 + 4 + 5", int64(15)},
		{"complex_bool", "(1 < 2) && (3 > 2) || false", true},
	}

	optimizer := &StaticExprOptimizer{}
	// Test optimization performance only
	b.Run("optimization_only", func(b *testing.B) {
		for _, bm := range benchmarks {
			b.Run(bm.name, func(b *testing.B) {
				exp, err := parser.ParseExpr(bm.expr)
				if err != nil {
					b.Fatal(err)
				}
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := optimizer.Optimize(exp, nil)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})

	// Test parsing and optimization performance
	b.Run("parse_and_optimize", func(b *testing.B) {
		for _, bm := range benchmarks {
			b.Run(bm.name, func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					exp, err := parser.ParseExpr(bm.expr)
					if err != nil {
						b.Fatal(err)
					}
					_, err = optimizer.Optimize(exp, nil)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})

	// Test full compilation and optimization process
	b.Run("full_compile_and_optimize", func(b *testing.B) {
		compiler := &goExprCompiler{pretreatment: exprPretreatmentChain}
		for _, bm := range benchmarks {
			b.Run(bm.name, func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					expr, err := compiler.Compile(bm.expr)
					if err != nil {
						b.Fatal(err)
					}
					result, err := expr.Execute(nil)
					if err != nil {
						b.Fatal(err)
					}
					// Validate results
					var got interface{}
					switch result.Kind() {
					case reflect.Bool:
						got = result.Bool()
					case reflect.Int64:
						got = result.Int()
					case reflect.String:
						got = result.String()
					default:
						b.Fatalf("unexpected type: %v", result.Kind())
					}
					if got != bm.want {
						b.Fatalf("got %v, want %v", got, bm.want)
					}
				}
			})
		}
	})
}

// TestStaticExprOptimizer tests the correctness of static expression optimization
func TestStaticExprOptimizer(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected interface{}
	}{
		{"bool_eq", "1 == 1", true},
		{"bool_neq", "1 != 2", true},
		{"math_add", "1 + 2", int64(3)},
		{"math_mul", "2 * 3", int64(6)},
		{"math_complex", "10 + 20 * 3", int64(70)},
		{"string_concat", `"hello" + "world"`, "helloworld"},
		{"mixed_ops", "1 + 2 * 3 == 7", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Eval(tt.expr, nil)
			if err != nil {
				t.Fatalf("failed to eval expression: %v", err)
			}

			var actual interface{}
			switch result.Kind() {
			case reflect.Bool:
				actual = result.Bool()
			case reflect.Int64:
				actual = result.Int()
			case reflect.String:
				actual = result.String()
			default:
				t.Fatalf("unexpected result type: %v", result.Kind())
			}

			if actual != tt.expected {
				t.Errorf("got %v, want %v", actual, tt.expected)
			}
		})
	}
}
