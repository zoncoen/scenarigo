package template

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/zoncoen/scenarigo/internal/testutil"
)

func TestNew(t *testing.T) {
	tests := map[string]struct {
		str         string
		expectError bool
	}{
		"success": {
			str: `{{a.b[0]}}`,
		},
		"failed": {
			str:         `{{}`,
			expectError: true,
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			_, err := New(test.str)
			if !test.expectError && err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if test.expectError && err == nil {
				t.Fatal("expected error but got no error")
			}
		})
	}
}

func TestTemplate_Execute(t *testing.T) {
	tests := map[string]struct {
		str         string
		data        interface{}
		expect      interface{}
		expectError string
	}{
		"no parameter": {
			str:    "1",
			expect: "1",
		},
		"empty string": {
			str:    "",
			expect: "",
		},
		"empty parameter": {
			str:    "{{}}",
			expect: "",
		},
		"empty parameter with string": {
			str:    "prefix-{{}}-suffix",
			expect: "prefix--suffix",
		},
		"string": {
			str:    `{{"foo"}}`,
			expect: "foo",
		},
		"integer": {
			str:    "{{1}}",
			expect: 1,
		},
		"true": {
			str:    "{{true}}",
			expect: true,
		},
		"false": {
			str:    "{{false}}",
			expect: false,
		},
		"add": {
			str:    `foo-{{ "bar" + "-" + "baz" }}`,
			expect: "foo-bar-baz",
		},
		"query from data": {
			str: "{{a.b[1]}}",
			data: map[string]map[string][]string{
				"a": {
					"b": {"ng", "ok"},
				},
			},
			expect: "ok",
		},

		"function call": {
			str: `{{f("ok")}}`,
			data: map[string]func(string) string{
				"f": func(s string) string { return s },
			},
			expect: "ok",
		},
		"call function that have argument required cast": {
			str: `{{f(1, 2, 3, 4, 5)}}`,
			data: map[string]func(int, int8, int16, int32, int64) int{
				"f": func(a0 int, a1 int8, a2 int16, a3 int32, a4 int64) int {
					return a0 + int(a1) + int(a2) + int(a3) + int(a4)
				},
			},
			expect: 15,
		},
		"call function that have variadic arguments": {
			str: `{{f(1, 2, 3, 4, 5)}}`,
			data: map[string]func(int, ...float32) int{
				"f": func(a0 int, args ...float32) int {
					sum := a0
					for _, a := range args {
						sum += int(a)
					}
					return sum
				},
			},
			expect: 15,
		},
		"function call (with nil error)": {
			str: `{{f("ok")}}`,
			data: map[string]interface{}{
				"f": func(s string) (string, error) { return s, nil },
			},
			expect: "ok",
		},
		"invalid function argument": {
			str: `{{f(1, 2, 3)}}`,
			data: map[string]func(int, int) int{
				"f": func(a0, a1 int) int {
					return a0 + a1
				},
			},
			expectError: "expected function argument number is 2 but specified 3 arguments",
		},
		"invalid function argument ( variadic arguments )": {
			str: `{{f()}}`,
			data: map[string]func(int, ...float32) int{
				"f": func(a0 int, args ...float32) int {
					sum := a0
					for _, a := range args {
						sum += int(a)
					}
					return sum
				},
			},
			expectError: "too few arguments to function: expected minimum argument number is 1. but specified 0 arguments",
		},
		"invalid function argument type (ident)": {
			str: `{{fn("1")}}`,
			data: map[string]func(int){
				"fn": func(a int) {},
			},
			expectError: "can't use string as int in arguments[0] to fn",
		},
		"invalid function argument type (selector)": {
			str: `{{m.fn("1")}}`,
			data: map[string]interface{}{
				"m": map[string]func(int){
					"fn": func(a int) {},
				},
			},
			expectError: "can't use string as int in arguments[0] to fn",
		},
		"function call (second value is not an error)": {
			str: `{{f()}}`,
			data: map[string]interface{}{
				"f": func() (interface{}, interface{}) { return nil, error(nil) },
			},
			expectError: "second returned value must be an error",
		},
		"function call (with error)": {
			str: `{{f()}}`,
			data: map[string]interface{}{
				"f": func() (interface{}, error) { return nil, errors.New("f() error") },
			},
			expectError: "f() error",
		},

		"method call": {
			str: `{{s.Echo("a") + s.Repeat("b") + p.Echo("c") + p.Self().Repeat(d)}}`,
			data: map[string]interface{}{
				"s": echoStruct{},
				"p": &echoStruct{},
				"d": testutil.ToPtr("d"),
			},
			expect: "abbcdd",
		},
		"method not found": {
			str: `{{p.Invalid()}}`,
			data: map[string]interface{}{
				"p": &echoStruct{},
			},
			expectError: `failed to execute: {{p.Invalid()}}: ".Invalid" not found`,
		},
		"invalid method argument": {
			str: `{{p.Repeat(a)}}`,
			data: map[string]interface{}{
				"p": &echoStruct{},
				"a": 1.2,
			},
			expectError: "can't use float64 as string in arguments[0] to Repeat",
		},

		"left arrow func": {
			str: strings.Trim(`
{{echo <-}}:
  message: '{{message}}'
`, "\n"),
			data: map[string]interface{}{
				"echo":    &echoFunc{},
				"message": "hello",
			},
			expect: "hello",
		},
		"left arrow func with function in argument": {
			str: strings.Trim(`
{{exec <-}}: '{{f}}'
`, "\n"),
			data: map[string]interface{}{
				"exec": &execFunc{},
				"f":    func() string { return "hello" },
			},
			expect: "hello",
		},
		"left arrow func (nest)": {
			str: strings.Trim(`
{{join <-}}:
  prefix: preout-
  text: |-
    {{join <-}}:
      prefix: prein-
      text: '{{text}}'
      suffix: -sufin
  suffix: -sufout
`, "\n"),
			data: map[string]interface{}{
				"join": &joinFunc{},
				"f":    func(s string) string { return s },
				"text": "test",
			},
			expect: "preout-prein-test-sufin-sufout",
		},
		"left arrow func with the non-string argument": {
			str: strings.Trim(`
{{join <-}}: '{{arg}}'
`, "\n"),
			data: map[string]interface{}{
				"join": &joinFunc{},
				"arg": map[string]interface{}{
					"prefix": "pre-",
					"text":   "{{text}}",
					"suffix": "-suf",
				},
				"text": 0,
			},
			expect: "pre-0-suf",
		},
		"left arrow func (complex)": {
			str: strings.Trim(`
{{echo <-}}:
  message: |-
    {{join <-}}:
      prefix: pre-
      text: |-
        {{call <-}}:
          f: '{{f}}'
          arg: '{{text}}'
      suffix: -suf
`, "\n"),
			data: map[string]interface{}{
				"echo": &echoFunc{},
				"join": &joinFunc{},
				"call": &callFunc{},
				"f":    func(s string) string { return s },
				"text": "test",
			},
			expect: "pre-test-suf",
		},
		"not found": {
			str:         "{{a.b[1]}}",
			expectError: `".a.b[1]" not found`,
		},
		"panic": {
			str: "{{panic()}}",
			data: map[string]interface{}{
				"panic": func() { panic("omg") },
			},
			expectError: "omg",
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			tmpl, err := New(test.str)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			i, err := tmpl.Execute(test.data)
			if test.expectError == "" && err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if test.expectError != "" {
				if err == nil {
					t.Fatal("expected error but got no error")
				}
				if got, expected := err.Error(), test.expectError; !strings.Contains(got, expected) {
					t.Errorf("expected error %q but got %q", expected, got)
				}
			}
			if diff := cmp.Diff(test.expect, i); diff != "" {
				t.Errorf("diff: (-want +got)\n%s", diff)
			}
		})
	}
}

func TestLeftArrowFunctionArg(t *testing.T) {
	tests := map[string]struct {
		str    string
		data   map[string]interface{}
		expect interface{}
	}{
		"no template": {
			str: strings.TrimPrefix(`
a: 1
b: 2
`, "\n"),
			expect: map[string]interface{}{
				"a": uint64(1),
				"b": uint64(2),
			},
		},
		"string": {
			str:    `'{{"test"}}'`,
			expect: "test",
		},
		"int": {
			str:    `'{{1}}'`,
			expect: uint64(1),
		},
		"int string": {
			str:    `'{{"1"}}'`,
			expect: "1",
		},
		"map array": {
			str: strings.TrimPrefix(`
users:
  - '{{user}}'
`, "\n"),
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "Alice",
					"age":  20,
				},
			},
			expect: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{
						"name": "Alice",
						"age":  uint64(20),
					},
				},
			},
		},
		"map map": {
			str: strings.TrimPrefix(`
admin: '{{user}}'
`, "\n"),
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "Alice",
					"age":  20,
				},
			},
			expect: map[string]interface{}{
				"admin": map[string]interface{}{
					"name": "Alice",
					"age":  uint64(20),
				},
			},
		},
		"complex function call": {
			str: strings.TrimPrefix(`
prefix: pre-
text: |-
  {{call <-}}:
    f: '{{f}}'
    arg: '{{text}}'
suffix: -suf
`, "\n"),
			data: map[string]interface{}{
				"call": &callFunc{},
				"f":    func(s string) string { return s },
				"text": "test",
			},
			expect: map[string]interface{}{
				"prefix": "pre-",
				"text":   "test",
				"suffix": "-suf",
			},
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			lines := []string{"{{dump <-}}:"}
			for _, line := range strings.Split(test.str, "\n") {
				lines = append(lines, fmt.Sprintf("  %s", line))
			}
			tmpl, err := New(strings.Join(lines, "\n"))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			tmpl.executingLeftArrowExprArg = true
			data := test.data
			if data == nil {
				data = map[string]interface{}{
					"dump": &dumpFunc{},
				}
			} else {
				data["dump"] = &dumpFunc{}
			}
			v, err := tmpl.Execute(data)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			s, ok := v.(string)
			if !ok {
				t.Fatalf("expect string but got %T", v)
			}
			var i interface{}
			if err := yaml.Unmarshal([]byte(s), &i); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.expect, i); diff != "" {
				t.Errorf("diff: (-want +got)\n%s", diff)
			}
		})
	}
}

func TestTemplate_ExecuteDirect(t *testing.T) {
	tests := map[string]struct {
		i           interface{}
		data        interface{}
		expect      interface{}
		expectError bool
	}{
		"not found by MapItem": {
			i: yaml.MapSlice{
				{
					Key:   "a",
					Value: "{{b}}",
				},
			},
			expectError: true,
		},
		"not found by struct": {
			i: struct {
				A string
			}{
				A: "{{b}}",
			},
			expectError: true,
		},
		"not found by struct with tag": {
			i: struct {
				A string `yaml:"a"`
			}{
				A: "{{b}}",
			},
			expectError: true,
		},
		"not found by map": {
			i: map[string]string{
				"a": "{{b}}",
			},
			expectError: true,
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			i, err := Execute(test.i, test.data)
			if !test.expectError && err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if test.expectError && err == nil {
				t.Fatal("expected error but got no error")
			}
			if diff := cmp.Diff(test.expect, i); diff != "" {
				t.Errorf("diff: (-want +got)\n%s", diff)
			}
		})
	}
}

type echoStruct struct{}

func (s *echoStruct) Self() *echoStruct {
	return s
}

func (s echoStruct) Echo(str string) interface{} {
	return str
}

func (s *echoStruct) Repeat(str string) string {
	return strings.Repeat(str, 2)
}

var _ Func = &echoFunc{}

type echoFunc struct{}

type echoArg struct {
	Message string `yaml:"message"`
}

func (*echoFunc) Exec(in interface{}) (interface{}, error) {
	arg, ok := in.(echoArg)
	if !ok {
		return nil, errors.New("arg must be a echoArg")
	}
	return arg.Message, nil
}

func (*echoFunc) UnmarshalArg(unmarshal func(interface{}) error) (interface{}, error) {
	var arg echoArg
	if err := unmarshal(&arg); err != nil {
		return nil, err
	}
	return arg, nil
}

var _ Func = &execFunc{}

type execFunc struct{}

func (*execFunc) Exec(in interface{}) (interface{}, error) {
	v := reflect.ValueOf(in)
	if !v.IsValid() {
		return nil, errors.New("invalid value")
	}
	if v.Kind() != reflect.Func {
		return nil, errors.Errorf("arg must be a function: %v", in)
	}
	t := v.Type()
	if n := t.NumIn(); n != 0 {
		return nil, errors.Errorf("number of arguments must be 0 but got %d", n)
	}
	if n := t.NumOut(); n != 1 {
		return nil, errors.Errorf("number of arguments must be 1 but got %d", n)
	}
	return v.Call(nil)[0].Interface(), nil
}

func (*execFunc) UnmarshalArg(unmarshal func(interface{}) error) (interface{}, error) {
	var arg interface{}
	if err := unmarshal(&arg); err != nil {
		return nil, err
	}
	return arg, nil
}

type joinFunc struct{}

type joinArg struct {
	Prefix string `yaml:"prefix"`
	Text   string `yaml:"text"`
	Suffix string `yaml:"suffix"`
}

func (*joinFunc) Exec(in interface{}) (interface{}, error) {
	arg, ok := in.(*joinArg)
	if !ok {
		return nil, errors.New("arg must be a joinArg")
	}
	return arg.Prefix + arg.Text + arg.Suffix, nil
}

func (*joinFunc) UnmarshalArg(unmarshal func(interface{}) error) (interface{}, error) {
	var arg joinArg
	if err := unmarshal(&arg); err != nil {
		return nil, err
	}
	return &arg, nil
}

type callFunc struct{}

type callArg struct {
	F   interface{} `yaml:"f"`
	Arg string      `yaml:"arg"`
}

func (*callFunc) Exec(in interface{}) (interface{}, error) {
	arg, ok := in.(*callArg)
	if !ok {
		return nil, errors.New("arg must be a callArg")
	}
	f, ok := arg.F.(func(string) string)
	if !ok {
		return nil, errors.New("arg.f must be a func(string) string")
	}
	return f(arg.Arg), nil
}

func (*callFunc) UnmarshalArg(unmarshal func(interface{}) error) (interface{}, error) {
	var arg callArg
	if err := unmarshal(&arg); err != nil {
		return nil, err
	}
	return &arg, nil
}

func TestFuncStash(t *testing.T) {
	var s funcStash
	name := s.save("value")
	if s[name] != "value" {
		t.Fatal("failed to save")
	}
}

var _ Func = &dumpFunc{}

type dumpFunc struct{}

func (*dumpFunc) Exec(in interface{}) (interface{}, error) {
	return in, nil
}

func (*dumpFunc) UnmarshalArg(unmarshal func(interface{}) error) (interface{}, error) {
	var arg interface{}
	if err := unmarshal(&arg); err != nil {
		return nil, err
	}
	return arg, nil
}
