package cache

import (
	"context"
	"errors"
	"testing"
)

func TestInMemoryScopeCache_SetAndGet(t *testing.T) {
	c := InMemoryScopeCache()
	err := c.Set(context.Background(), "key", "value")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	var result string
	err = c.Get(context.Background(), "key", &result)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "value" {
		t.Errorf("Expected 'value', got '%s'", result)
	}

	// pointer type
	var result2 *string
	var value = "value2"
	err = c.Set(context.Background(), "key2", &value)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	err = c.Get(context.Background(), "key2", &result2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if *result2 != "value2" {
		t.Errorf("Expected 'value2', got '%s'", *result2)
	}

	// struct type
	type T struct {
		A int
		B string
	}
	var result3 = T{
		A: 1,
		B: "b",
	}
	if err = c.Set(context.Background(), "key3", result3); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	var result4 T
	if err = c.Get(context.Background(), "key3", &result4); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result4 != result3 {
		t.Errorf("Expected {1, 'b'}, got %v", result4)
	}

	// struct pointer type
	var result5 *T
	if err = c.Set(context.Background(), "key4", &result3); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if err = c.Get(context.Background(), "key4", &result5); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if *result5 != result3 {
		t.Errorf("Expected {1, 'b'}, got %v", *result5)
	}

	// if original value is changed, the cache value should not be changed
	result3.A = 2
	result3.B = "c"
	if err = c.Get(context.Background(), "key4", &result5); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result5.B != "b" && result5.A != 1 {
		t.Errorf("Expected {1, 'b'}, got %v", *result5)
	}

	// original value is prt, but cache value is not
	var result6 *T
	if err = c.Set(context.Background(), "key5", &result3); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if err = c.Get(context.Background(), "key5", &result6); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	t.Log(result6)
}

func TestInMemoryScopeCache_GetNonExistentKey(t *testing.T) {
	c := InMemoryScopeCache()
	var result string
	err := c.Get(context.Background(), "nonexistent", &result)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if !errors.Is(err, ErrCacheNotFound) {
		t.Errorf("Expected ErrCacheNotFound, got %v", err)
	}
}

func TestInMemoryScopeCache_Flush(t *testing.T) {
	c := InMemoryScopeCache()
	err := c.Set(context.Background(), "key", "value")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	err = c.Flush(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	var result string
	err = c.Get(context.Background(), "key", &result)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if !errors.Is(err, ErrCacheNotFound) {
		t.Errorf("Expected ErrCacheNotFound, got %v", err)
	}
}

func BenchmarkInMemoryScopeCache_Set(b *testing.B) {
	c := InMemoryScopeCache()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		_ = c.Set(ctx, "key", "value")
	}
}

func BenchmarkInMemoryScopeCache_Get(b *testing.B) {
	c := InMemoryScopeCache()
	ctx := context.Background()
	_ = c.Set(ctx, "key", "value")
	var result string
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Get(ctx, "key", &result)
	}
}

func BenchmarkInMemoryScopeCache_Flush(b *testing.B) {
	c := InMemoryScopeCache()
	ctx := context.Background()
	_ = c.Set(ctx, "key", "value")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Flush(ctx)
	}
}
