package reflectutil

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestConvertStringsMap(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tests := map[string]struct {
			v      interface{}
			expect map[string][]string
		}{
			"map[string]string": {
				v:      map[string]string{"A": "a"},
				expect: map[string][]string{"A": {"a"}},
			},
			"map[string][]string": {
				v:      map[string][]string{"A": {"a"}},
				expect: map[string][]string{"A": {"a"}},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				got, err := ConvertStringsMap(reflect.ValueOf(test.v))
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if diff := cmp.Diff(test.expect, got); diff != "" {
					t.Fatalf("differs: (-want +got)\n%s", diff)
				}
			})
		}
	})
	t.Run("error", func(t *testing.T) {
		tests := map[string]struct {
			v interface{}
		}{
			"nil": {
				v: nil,
			},
			"int": {
				v: 555,
			},
			"map[int]int": {
				v: map[int]int{0: 0},
			},
			"map[string]int": {
				v: map[string]int{"0": 0},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				if _, err := ConvertStringsMap(reflect.ValueOf(test.v)); err == nil {
					t.Fatal("expected error but no error")
				}
			})
		}
	})
}

func TestConvertStrings(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tests := map[string]struct {
			v      interface{}
			expect []string
		}{
			"string": {
				v:      "test",
				expect: []string{"test"},
			},
			"[]string": {
				v:      []string{"1", "2"},
				expect: []string{"1", "2"},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				got, err := convertStrings(reflect.ValueOf(test.v))
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if diff := cmp.Diff(test.expect, got); diff != "" {
					t.Fatalf("differs: (-want +got)\n%s", diff)
				}
			})
		}
	})
	t.Run("error", func(t *testing.T) {
		tests := map[string]struct {
			v interface{}
		}{
			"nil": {
				v: nil,
			},
			"int": {
				v: 555,
			},
			"nil (string pointer)": {
				v: (*string)(nil),
			},
			"[]int": {
				v: []int{1},
			},
			"[]interface{}": {
				v: []interface{}{nil},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				if _, err := convertStrings(reflect.ValueOf(test.v)); err == nil {
					t.Fatal("expected error but no error")
				}
			})
		}
	})
}
