package template

import (
	"reflect"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
)

func TestExecute(t *testing.T) {
	type s struct {
		Str        string
		StrPtr     *string
		unexported string
	}
	var (
		tmpl              = `{{"test"}}`
		str               = "test"
		iface interface{} = `{{"test"}}`
	)
	tests := map[string]struct {
		in       interface{}
		expected interface{}
		vars     interface{}
	}{
		"string": {
			in:       "test",
			expected: "test",
		},
		"*string": {
			in:       &str,
			expected: &str,
		},
		"template string": {
			in:       "{{test}}",
			expected: "test",
			vars: map[string]string{
				"test": "test",
			},
		},
		"integer": {
			in:       1,
			expected: 1,
		},
		"nil": {
			in:       nil,
			expected: nil,
		},
		"nil map": {
			in:       map[interface{}]interface{}(nil),
			expected: map[interface{}]interface{}(nil),
		},
		"nil slice": {
			in:       []interface{}(nil),
			expected: []interface{}(nil),
		},
		"map[string]string": {
			in: map[string]string{
				"env": `{{"test"}}`,
			},
			expected: map[string]string{
				"env": "test",
			},
		},
		"map[string]*string": {
			in: map[string]*string{
				"env": &tmpl,
			},
			expected: map[string]*string{
				"env": &str,
			},
		},
		"map[string]interface{}": {
			in: map[string]interface{}{
				"env":     `{{"test"}}`,
				"version": "{{1}}",
				"nil":     nil,
			},
			expected: map[string]interface{}{
				"env":     "test",
				"version": 1,
				"nil":     nil,
			},
		},
		"map[string][]string": {
			in: map[string][]string{
				"env": {`{{"test"}}`},
			},
			expected: map[string][]string{
				"env": {"test"},
			},
		},
		"[]string": {
			in:       []string{`{{"one"}}`, "two", `{{"three"}}`},
			expected: []string{"one", "two", "three"},
		},
		"[]*string": {
			in:       []*string{&tmpl},
			expected: []*string{&str},
		},
		"[]interface{}": {
			in:       []interface{}{`{{"one"}}`, `{{1}}`, nil},
			expected: []interface{}{"one", 1, nil},
		},
		"yaml.MapSlice": {
			in: yaml.MapSlice{
				yaml.MapItem{
					Key:   "id",
					Value: 100,
				},
				yaml.MapItem{
					Key:   "name",
					Value: `{{"Bob"}}`,
				},
			},
			expected: yaml.MapSlice{
				yaml.MapItem{
					Key:   "id",
					Value: 100,
				},
				yaml.MapItem{
					Key:   "name",
					Value: "Bob",
				},
			},
		},
		"yaml.MapSlice (Value is nil)": {
			in: yaml.MapSlice{
				yaml.MapItem{
					Key:   "key",
					Value: nil,
				},
			},
			expected: yaml.MapSlice{
				yaml.MapItem{
					Key:   "key",
					Value: nil,
				},
			},
		},
		"struct": {
			in: s{
				Str:        tmpl,
				StrPtr:     &tmpl,
				unexported: tmpl,
			},
			expected: s{
				Str:        str,
				StrPtr:     &str,
				unexported: tmpl, // can't set a value to unexported fileld
			},
		},
		"struct pointer": {
			in: &s{
				Str:        tmpl,
				StrPtr:     &tmpl,
				unexported: tmpl,
			},
			expected: &s{
				Str:        str,
				StrPtr:     &str,
				unexported: tmpl, // can't set a value to unexported fileld
			},
		},
		"pointer to interface{}": {
			in:       &iface,
			expected: "test",
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			got, err := Execute(test.in, test.vars)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(test.expected, got, cmp.AllowUnexported(s{})); diff != "" {
				t.Errorf("differs: (-want +got)\n%s", diff)
			}
		})
	}
}

func TestConvert(t *testing.T) {
	convertToStr := convert(reflect.TypeOf(""))
	convertToStrPtr := convert(reflect.PtrTo(reflect.TypeOf("")))
	t.Run("convert to string", func(t *testing.T) {
		s := "test"
		v, err := convertToStr(reflect.ValueOf(&s), nil)
		if err != nil {
			t.Fatalf("failed to convert: %s", err)
		}
		if got, expect := v.Type().Kind(), reflect.String; got != expect {
			t.Fatalf("expect %s but got %s", expect, got)
		}
	})
	t.Run("convert to string pointer", func(t *testing.T) {
		v, err := convertToStrPtr(reflect.ValueOf("test"), nil)
		if err != nil {
			t.Fatalf("failed to convert: %s", err)
		}
		if got, expect := v.Type().Kind(), reflect.Ptr; got != expect {
			t.Fatalf("expect %s but got %s", expect, got)
		}
		if got, expect := v.Type().Elem().Kind(), reflect.String; got != expect {
			t.Fatalf("expect %s but got %s", expect, got)
		}
	})
	t.Run("can't convert", func(t *testing.T) {
		v, err := convertToStrPtr(reflect.ValueOf(1), nil)
		if err != nil {
			t.Fatalf("failed to convert: %s", err)
		}
		if got, expect := v.Type().Kind(), reflect.Int; got != expect {
			t.Fatalf("expect %s but got %s", expect, got)
		}
	})
	t.Run("error", func(t *testing.T) {
		_, err := convertToStrPtr(reflect.Value{}, errors.New("execute() failed"))
		if err == nil {
			t.Fatal("no error")
		}
	})
}
