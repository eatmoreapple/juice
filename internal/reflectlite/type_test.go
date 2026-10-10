package reflectlite

import (
	"reflect"
	"slices"
	"testing"
)

func TestIndirectType_type_test(t *testing.T) {
	type myInt int

	tests := []struct {
		name string
		in   reflect.Type
		want reflect.Type
	}{
		{"non-pointer", reflect.TypeFor[myInt](), reflect.TypeFor[myInt]()},
		{"pointer", reflect.TypeFor[*myInt](), reflect.TypeFor[myInt]()},
		{"double pointer", reflect.TypeFor[**myInt](), reflect.TypeFor[myInt]()},
		{"nil", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IndirectType(tt.in); got != tt.want {
				t.Errorf("IndirectType(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestLookupFieldIndexByTag_type_test(t *testing.T) {
	type InnerMost struct {
		DeepField string `tag:"deep"`
	}
	type Inner struct {
		InnerMost     // Anonymous
		MidField  int `tag:"mid"`
	}
	type Outer struct {
		Inner           // Anonymous
		OuterField bool `tag:"outer"`
	}

	rtOuter := reflect.TypeFor[Outer]()

	tests := []struct {
		name  string
		rType reflect.Type
		value string
		want  []int
		ok    bool
	}{
		{"direct field", rtOuter, "outer", []int{1}, true},
		{"embedded field", rtOuter, "mid", []int{0, 1}, true},
		{"deeply embedded field", rtOuter, "deep", []int{0, 0, 0}, true},
		{"pointer to struct", reflect.TypeFor[*Outer](), "outer", []int{1}, true},
		{"not found", rtOuter, "nope", nil, false},
		{"non-struct", reflect.TypeFor[int](), "outer", nil, false},
		{"nil type", nil, "outer", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := LookupFieldIndexByTag(tt.rType, "tag", tt.value)
			if ok != tt.ok {
				t.Fatalf("LookupFieldIndexByTag(%v, tag, %q) ok = %v, want %v", tt.rType, tt.value, ok, tt.ok)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("LookupFieldIndexByTag(%v, tag, %q) = %v, want %v", tt.rType, tt.value, got, tt.want)
			}
		})
	}
}

// TestLookupFieldIndexByTagSkipsUnexported_type_test verifies that the lookup
// never returns the index path of an unexported field. Such a field's
// reflect.Value is read-only and cannot be passed to Interface(), which would
// panic in callers that render parameters.
func TestLookupFieldIndexByTagSkipsUnexported_type_test(t *testing.T) {
	type embedded struct {
		Promoted int `tag:"promoted"` // exported field promoted via unexported embedded struct
	}
	type inner struct { //nolint:unused // referenced via reflection only
		Inner int `tag:"inner"`
	}
	type Outer struct {
		embedded
		named  inner  //nolint:unused // named unexported struct field
		secret string `tag:"secret"` //nolint:unused // unexported leaf
	}

	rt := reflect.TypeFor[Outer]()

	tests := []struct {
		name  string
		value string
		want  []int
		ok    bool
	}{
		{"unexported leaf is skipped", "secret", nil, false},
		{"field of unexported named struct is skipped", "inner", nil, false},
		{"exported field promoted through unexported embedded struct", "promoted", []int{0, 0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := LookupFieldIndexByTag(rt, "tag", tt.value)
			if ok != tt.ok {
				t.Fatalf("LookupFieldIndexByTag(%v, tag, %q) ok = %v, want %v", rt, tt.value, ok, tt.ok)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("LookupFieldIndexByTag(%v, tag, %q) = %v, want %v", rt, tt.value, got, tt.want)
			}
		})
	}
}
