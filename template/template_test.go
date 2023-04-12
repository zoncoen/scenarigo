package template

import (
	"fmt"
	"math"
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
	tests := map[string]executeTestCase{
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
			expect: int64(1),
		},
		"float": {
			str:    "{{1.23}}",
			expect: 1.23,
		},
		"true": {
			str:    "{{true}}",
			expect: true,
		},
		"false": {
			str:    "{{false}}",
			expect: false,
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
	runExecute(t, tests)
}

func TestTemplate_Execute_UnaryExpr(t *testing.T) {
	tests := map[string]executeTestCase{
		"negative int": {
			str: "{{-v}}",
			data: map[string]interface{}{
				"v": math.MaxInt,
			},
			expect: -math.MaxInt,
		},
		"-int greater than max int": {
			str: "{{-v}}",
			data: map[string]interface{}{
				"v": math.MinInt,
			},
			expectError: fmt.Sprintf("failed to execute: {{-v}}: -(%d) overflows int", math.MinInt),
		},
		"negative int8": {
			str: "{{-v}}",
			data: map[string]interface{}{
				"v": int8(math.MaxInt8),
			},
			expect: int8(-math.MaxInt8),
		},
		"-int8 greater than max int8": {
			str: "{{-v}}",
			data: map[string]interface{}{
				"v": int8(math.MinInt8),
			},
			expectError: fmt.Sprintf("failed to execute: {{-v}}: -(%d) overflows int8", math.MinInt8),
		},
		"negative int16": {
			str: "{{-v}}",
			data: map[string]interface{}{
				"v": int16(math.MaxInt16),
			},
			expect: int16(-math.MaxInt16),
		},
		"-int16 greater than max int16": {
			str: "{{-v}}",
			data: map[string]interface{}{
				"v": int16(math.MinInt16),
			},
			expectError: fmt.Sprintf("failed to execute: {{-v}}: -(%d) overflows int16", math.MinInt16),
		},
		"negative int32": {
			str: "{{-v}}",
			data: map[string]interface{}{
				"v": int32(math.MaxInt32),
			},
			expect: int32(-math.MaxInt32),
		},
		"-int32 greater than max int32": {
			str: "{{-v}}",
			data: map[string]interface{}{
				"v": int32(math.MinInt32),
			},
			expectError: fmt.Sprintf("failed to execute: {{-v}}: -(%d) overflows int32", math.MinInt32),
		},
		"negative int64": {
			str: "{{-v}}",
			data: map[string]interface{}{
				"v": int64(math.MaxInt64),
			},
			expect: int64(-math.MaxInt64),
		},
		"-int64 greater than max int64": {
			str: "{{-v}}",
			data: map[string]interface{}{
				"v": int64(math.MinInt64),
			},
			expectError: fmt.Sprintf("failed to execute: {{-v}}: -(%d) overflows int64", math.MinInt64),
		},
		"negative uint": {
			str: "{{-v}}",
			data: map[string]interface{}{
				"v": uint(math.MaxInt),
			},
			expect: -math.MaxInt,
		},
		"-uint less than min int": {
			str: "{{-v}}",
			data: map[string]interface{}{
				"v": uint(math.MaxUint),
			},
			expectError: fmt.Sprintf("failed to execute: {{-v}}: -%d overflows int", uint(math.MaxUint)),
		},
		"negative uint8": {
			str: "{{-v}}",
			data: map[string]interface{}{
				"v": uint8(1),
			},
			expect: -1,
		},
		"negative uint16": {
			str: "{{-v}}",
			data: map[string]interface{}{
				"v": uint16(1),
			},
			expect: -1,
		},
		"negative uint32": {
			str: "{{-v}}",
			data: map[string]interface{}{
				"v": uint32(1),
			},
			expect: -1,
		},
		"negative uint64": {
			str: "{{-v}}",
			data: map[string]interface{}{
				"v": uint64(1),
			},
			expect: -1,
		},
		"-uint64 less than min int": {
			str: "{{-v}}",
			data: map[string]interface{}{
				"v": uint64(math.MaxUint64),
			},
			expectError: fmt.Sprintf("failed to execute: {{-v}}: -%d overflows int", uint64(math.MaxUint64)),
		},
		"negative float32": {
			str: "{{-v}}",
			data: map[string]interface{}{
				"v": float32(math.MaxFloat32),
			},
			expect: -float32(math.MaxFloat32),
		},
		"negative float64": {
			str: "{{-v}}",
			data: map[string]interface{}{
				"v": float64(math.MaxFloat64),
			},
			expect: -float64(math.MaxFloat64),
		},
		"!true": {
			str:    "{{!true}}",
			expect: false,
		},
		"!1": {
			str:         "{{!1}}",
			expectError: `failed to execute: {{!1}}: unknown operation: operator ! not defined on int64`,
		},
		"defined": {
			str: "{{defined(a.b)}}",
			data: map[string]any{
				"a": map[string]any{
					"b": `{{test}}`,
				},
			},
			expect: true,
		},
		"not defined": {
			str:    "{{defined(a.b)}}",
			expect: false,
		},
		"invalid argument to defined()": {
			str:         "{{defined(true)}}",
			expectError: "failed to execute: {{defined(true)}}: invalid argument to defined()",
		},
	}
	runExecute(t, tests)
}

func TestTemplate_Execute_BinaryExpr(t *testing.T) {
	t.Run("+", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"add ints": {
				str: `{{v + int(1)}}`,
				data: map[string]interface{}{
					"v": int64(math.MaxInt64 - 1),
				},
				expect: int64(math.MaxInt64),
			},
			"add negative ints": {
				str: `{{v + int(-1)}}`,
				data: map[string]interface{}{
					"v": int64(math.MinInt64 + 1),
				},
				expect: int64(math.MinInt64),
			},
			"add ints (greater than max int64)": {
				str: `{{v + int(1)}}`,
				data: map[string]interface{}{
					"v": int64(math.MaxInt64),
				},
				expectError: "failed to execute: {{v + int(1)}}: invalid operation: 9223372036854775807 + 1 overflows int",
			},
			"add ints (less than min int64)": {
				str: `{{v + int(-1)}}`,
				data: map[string]interface{}{
					"v": int64(math.MinInt64),
				},
				expectError: "failed to execute: {{v + int(-1)}}: invalid operation: -9223372036854775808 + -1 overflows int",
			},
			"add uints": {
				str:    `{{uint(1) + uint(2)}}`,
				expect: uint64(3),
			},
			"add uints (greater than max uint64)": {
				str: `{{v + uint(1)}}`,
				data: map[string]interface{}{
					"v": uint64(math.MaxUint64),
				},
				expectError: "failed to execute: {{v + uint(1)}}: invalid operation: 18446744073709551615 + 1 overflows uint",
			},
			"add floats": {
				str:    `{{1.0 + 0.23}}`,
				expect: 1.23,
			},
			"add strings": {
				str:    `foo-{{ "bar" + "-" + "baz" }}`,
				expect: "foo-bar-baz",
			},
			"failed to add mismatched types": {
				str:         `{{1 + uint(1)}}`,
				expectError: "failed to execute: {{1 + uint(1)}}: invalid operation: 1 + 0x1: mismatched types int64 and uint",
			},
			"failed to add structs": {
				str:         `{{true + false}}`,
				expectError: "failed to execute: {{true + false}}: invalid operation: operator + not defined on true (value of type bool)",
			},
			"failed to add untyped nils": {
				str: `{{v + v}}`,
				data: map[string]interface{}{
					"v": nil,
				},
				expectError: "failed to execute: {{v + v}}: invalid operation: operator + not defined on nil",
			},
		}
		runExecute(t, tests)
	})

	t.Run("-", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"sub positive int": {
				str: `{{v - int(1)}}`,
				data: map[string]interface{}{
					"v": int64(math.MinInt64 + 1),
				},
				expect: int64(math.MinInt64),
			},
			"sub negative int": {
				str: `{{v - int(-1)}}`,
				data: map[string]interface{}{
					"v": int64(math.MaxInt64 - 1),
				},
				expect: int64(math.MaxInt64),
			},
			"sub ints (greater than max int64)": {
				str: `{{v - int(-1)}}`,
				data: map[string]interface{}{
					"v": int64(math.MaxInt64),
				},
				expectError: "failed to execute: {{v - int(-1)}}: invalid operation: 9223372036854775807 - -1 overflows int",
			},
			"sub ints (less than min int64)": {
				str: `{{v - int(1)}}`,
				data: map[string]interface{}{
					"v": int64(math.MinInt64),
				},
				expectError: "failed to execute: {{v - int(1)}}: invalid operation: -9223372036854775808 - 1 overflows int",
			},
			"sub uints": {
				str:    `{{uint(2) - uint(1)}}`,
				expect: uint64(1),
			},
			"sub uints (less than 0)": {
				str:         `{{uint(1) - uint(2)}}`,
				expectError: "failed to execute: {{uint(1) - uint(2)}}: invalid operation: 1 - 2 overflows uint",
			},
			"sub floats": {
				str:    `{{1.0 - 0.23}}`,
				expect: 0.77,
			},
			"failed to sub mismatched types": {
				str:         `{{1 - uint(1)}}`,
				expectError: "failed to execute: {{1 - uint(1)}}: invalid operation: 1 - 0x1: mismatched types int64 and uint",
			},
			"failed to sub strings": {
				str:         `{{"a" - "b"}}`,
				expectError: `failed to execute: {{"a" - "b"}}: invalid operation: operator - not defined on "a" (value of type string)`,
			},
			"failed to sub untyped nils": {
				str: `{{v - v}}`,
				data: map[string]interface{}{
					"v": nil,
				},
				expectError: "failed to execute: {{v - v}}: invalid operation: operator - not defined on nil",
			},
		}
		runExecute(t, tests)
	})

	t.Run("*", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"mul ints": {
				str:    `{{2 * 3}}`,
				expect: int64(6),
			},
			"max int * -1": {
				str: `{{v * int(-1)}}`,
				data: map[string]interface{}{
					"v": int64(math.MaxInt64),
				},
				expect: int64(-math.MaxInt64),
			},
			"-1 * max int": {
				str: `{{int(-1) * v}}`,
				data: map[string]interface{}{
					"v": int64(math.MaxInt64),
				},
				expect: int64(-math.MaxInt64),
			},
			"int * int (greater than max int64)": {
				str: `{{v * int(2)}}`,
				data: map[string]interface{}{
					"v": int64(math.MaxInt64),
				},
				expectError: "failed to execute: {{v * int(2)}}: invalid operation: 9223372036854775807 * 2 overflows int",
			},
			"negative int * negative int (greater than max int64)": {
				str: `{{v * int(-2)}}`,
				data: map[string]interface{}{
					"v": int64(math.MinInt64),
				},
				expectError: "failed to execute: {{v * int(-2)}}: invalid operation: -9223372036854775808 * -2 overflows int",
			},
			"int * negative int (less than min int64)": {
				str: `{{v * int(-2)}}`,
				data: map[string]interface{}{
					"v": int64(math.MaxInt64),
				},
				expectError: "failed to execute: {{v * int(-2)}}: invalid operation: 9223372036854775807 * -2 overflows int",
			},
			"negative int * int (less than min int64)": {
				str: `{{int(-2) * v}}`,
				data: map[string]interface{}{
					"v": int64(math.MaxInt64),
				},
				expectError: "failed to execute: {{int(-2) * v}}: invalid operation: -2 * 9223372036854775807 overflows int",
			},
			"mul uints": {
				str:    `{{uint(2) * uint(3)}}`,
				expect: uint64(6),
			},
			"max uint * uint (greater than max uint64)": {
				str: `{{v * uint(2)}}`,
				data: map[string]interface{}{
					"v": uint64(math.MaxUint64),
				},
				expectError: "failed to execute: {{v * uint(2)}}: invalid operation: 18446744073709551615 * 2 overflows uint",
			},
			"uint * max uint (greater than max uint64)": {
				str: `{{uint(2) * v}}`,
				data: map[string]interface{}{
					"v": uint64(math.MaxUint64),
				},
				expectError: "failed to execute: {{uint(2) * v}}: invalid operation: 2 * 18446744073709551615 overflows uint",
			},
			"mul floats": {
				str:    `{{1.2 * 3.4}}`,
				expect: float64(4.08),
			},
			"failed to mul mismatched types": {
				str:         `{{1 * uint(1)}}`,
				expectError: "failed to execute: {{1 * uint(1)}}: invalid operation: 1 * 0x1: mismatched types int64 and uint",
			},
			"failed to mul strings": {
				str:         `{{"a" * "b"}}`,
				expectError: `failed to execute: {{"a" * "b"}}: invalid operation: operator * not defined on "a" (value of type string)`,
			},
		}
		runExecute(t, tests)
	})

	t.Run("/", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"quo ints": {
				str:    `{{3 / -2}}`,
				expect: int64(-1),
			},
			"division int by 0": {
				str:         `{{1 / 0}}`,
				expectError: "failed to execute: {{1 / 0}}: invalid operation: division by 0",
			},
			"quo uints": {
				str:    `{{uint(3) / uint(2)}}`,
				expect: uint64(1),
			},
			"division uint by 0": {
				str:         `{{uint(1) / uint(0)}}`,
				expectError: "failed to execute: {{uint(1) / uint(0)}}: invalid operation: division by 0",
			},
			"quo floats": {
				str:    `{{float(3) / float(2)}}`,
				expect: float64(1.5),
			},
			"division float by 0": {
				str:         `{{float(1) / float(0)}}`,
				expectError: "failed to execute: {{float(1) / float(0)}}: invalid operation: division by 0",
			},
			"failed to quo mismatched types": {
				str:         `{{1 / uint(1)}}`,
				expectError: "failed to execute: {{1 / uint(1)}}: invalid operation: 1 / 0x1: mismatched types int64 and uint",
			},
			"failed to quo strings": {
				str:         `{{"a" / "b"}}`,
				expectError: `failed to execute: {{"a" / "b"}}: invalid operation: operator / not defined on "a" (value of type string)`,
			},
		}
		runExecute(t, tests)
	})

	t.Run("%", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"rem ints": {
				str:    `{{3 % -2}}`,
				expect: int64(1),
			},
			"division int by 0": {
				str:         `{{1 % 0}}`,
				expectError: "failed to execute: {{1 % 0}}: invalid operation: division by 0",
			},
			"rem uints": {
				str:    `{{uint(3) % uint(2)}}`,
				expect: uint64(1),
			},
			"division uint by 0": {
				str:         `{{uint(1) % uint(0)}}`,
				expectError: "failed to execute: {{uint(1) % uint(0)}}: invalid operation: division by 0",
			},
			"failed to rem mismatched types": {
				str:         `{{1 % uint(1)}}`,
				expectError: "failed to execute: {{1 % uint(1)}}: invalid operation: 1 % 0x1: mismatched types int64 and uint",
			},
			"failed to rem strings": {
				str:         `{{"a" % "b"}}`,
				expectError: `failed to execute: {{"a" % "b"}}: invalid operation: operator % not defined on "a" (value of type string)`,
			},
		}
		runExecute(t, tests)
	})

	t.Run("&&", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"true&&true": {
				str:    `{{true&&true}}`,
				expect: true,
			},
			"true&&false": {
				str:    `{{true&&false}}`,
				expect: false,
			},
		}
		runExecute(t, tests)
	})

	t.Run("||", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"true||false": {
				str:    `{{true||false}}`,
				expect: true,
			},
			"false||false": {
				str:    `{{false||false}}`,
				expect: false,
			},
		}
		runExecute(t, tests)
	})

	t.Run("==", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"true==true": {
				str:    `{{true==true}}`,
				expect: true,
			},
			"true==false": {
				str:    `{{true==false}}`,
				expect: false,
			},
			"nil==nil": {
				str: `{{v==v}}`,
				data: map[string]interface{}{
					"v": nil,
				},
				expect: true,
			},
		}
		runExecute(t, tests)
	})

	t.Run("!=", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"true!=true": {
				str:    `{{true!=true}}`,
				expect: false,
			},
			"true!=false": {
				str:    `{{true!=false}}`,
				expect: true,
			},
			"nil!=nil": {
				str: `{{v!=v}}`,
				data: map[string]interface{}{
					"v": nil,
				},
				expect: false,
			},
		}
		runExecute(t, tests)
	})

	t.Run("<", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"1<2": {
				str:    `{{1<2}}`,
				expect: true,
			},
			"1<1": {
				str:    `{{1<1}}`,
				expect: false,
			},
			"2<1": {
				str:    `{{2<1}}`,
				expect: false,
			},
			"uint(1)<uint(2)": {
				str:    `{{uint(1)<uint(2)}}`,
				expect: true,
			},
			"uint(1)<uint(1)": {
				str:    `{{uint(1)<uint(1)}}`,
				expect: false,
			},
			"uint(2)<uint(1)": {
				str:    `{{uint(2)<uint(1)}}`,
				expect: false,
			},
			"1.1<2.2": {
				str:    `{{1.1<2.2}}`,
				expect: true,
			},
			"1.1<1.1": {
				str:    `{{1.1<1.1}}`,
				expect: false,
			},
			"2.2<1.1": {
				str:    `{{2.2<1.1}}`,
				expect: false,
			},
			`"a"<"b"`: {
				str:    `{{"a"<"b"}}`,
				expect: true,
			},
			`"a"<"a"`: {
				str:    `{{"a"<"a"}}`,
				expect: false,
			},
			`"b"<"a"`: {
				str:    `{{"b"<"a"}}`,
				expect: false,
			},
		}
		runExecute(t, tests)
	})

	t.Run("<=", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"1<=2": {
				str:    `{{1<=2}}`,
				expect: true,
			},
			"1<=1": {
				str:    `{{1<=1}}`,
				expect: true,
			},
			"2<=1": {
				str:    `{{2<=1}}`,
				expect: false,
			},
			"uint(1)<=uint(2)": {
				str:    `{{uint(1)<=uint(2)}}`,
				expect: true,
			},
			"uint(1)<=uint(1)": {
				str:    `{{uint(1)<=uint(1)}}`,
				expect: true,
			},
			"uint(2)<=uint(1)": {
				str:    `{{uint(2)<=uint(1)}}`,
				expect: false,
			},
			"1.1<=2.2": {
				str:    `{{1.1<=2.2}}`,
				expect: true,
			},
			"1.1<=1.1": {
				str:    `{{1.1<=1.1}}`,
				expect: true,
			},
			"2.2<=1.1": {
				str:    `{{2.2<=1.1}}`,
				expect: false,
			},
			`"a"<="b"`: {
				str:    `{{"a"<="b"}}`,
				expect: true,
			},
			`"a"<="a"`: {
				str:    `{{"a"<="a"}}`,
				expect: true,
			},
			`"b"<="a"`: {
				str:    `{{"b"<="a"}}`,
				expect: false,
			},
		}
		runExecute(t, tests)
	})

	t.Run(">", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"1>2": {
				str:    `{{1>2}}`,
				expect: false,
			},
			"1>1": {
				str:    `{{1>1}}`,
				expect: false,
			},
			"2>1": {
				str:    `{{2>1}}`,
				expect: true,
			},
			"uint(1)>uint(2)": {
				str:    `{{uint(1)>uint(2)}}`,
				expect: false,
			},
			"uint(1)>uint(1)": {
				str:    `{{uint(1)>uint(1)}}`,
				expect: false,
			},
			"uint(2)>uint(1)": {
				str:    `{{uint(2)>uint(1)}}`,
				expect: true,
			},
			"1.1>2.2": {
				str:    `{{1.1>2.2}}`,
				expect: false,
			},
			"1.1>1.1": {
				str:    `{{1.1>1.1}}`,
				expect: false,
			},
			"2.2>1.1": {
				str:    `{{2.2>1.1}}`,
				expect: true,
			},
			`"a">"b"`: {
				str:    `{{"a">"b"}}`,
				expect: false,
			},
			`"a">"a"`: {
				str:    `{{"a">"a"}}`,
				expect: false,
			},
			`"b">"a"`: {
				str:    `{{"b">"a"}}`,
				expect: true,
			},
		}
		runExecute(t, tests)
	})

	t.Run(">=", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"1>=2": {
				str:    `{{1>=2}}`,
				expect: false,
			},
			"1>=1": {
				str:    `{{1>=1}}`,
				expect: true,
			},
			"2>=1": {
				str:    `{{2>=1}}`,
				expect: true,
			},
			"uint(1)>=uint(2)": {
				str:    `{{uint(1)>=uint(2)}}`,
				expect: false,
			},
			"uint(1)>=uint(1)": {
				str:    `{{uint(1)>=uint(1)}}`,
				expect: true,
			},
			"uint(2)>=uint(1)": {
				str:    `{{uint(2)>=uint(1)}}`,
				expect: true,
			},
			"1.1>=2.2": {
				str:    `{{1.1>=2.2}}`,
				expect: false,
			},
			"1.1>=1.1": {
				str:    `{{1.1>=1.1}}`,
				expect: true,
			},
			"2.2>=1.1": {
				str:    `{{2.2>=1.1}}`,
				expect: true,
			},
			`"a">="b"`: {
				str:    `{{"a">="b"}}`,
				expect: false,
			},
			`"a">="a"`: {
				str:    `{{"a">="a"}}`,
				expect: true,
			},
			`"b">="a"`: {
				str:    `{{"b">="a"}}`,
				expect: true,
			},
		}
		runExecute(t, tests)
	})

	t.Run("? :", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"true ? 1 : 2": {
				str:    `{{true ? 1 : 2}}`,
				expect: int64(1),
			},
			"false ? 1 : 2": {
				str:    `{{false ? 1 : 2}}`,
				expect: int64(2),
			},
			"1 ? 1 : 2": {
				str:         `{{1 ? 1 : 2}}`,
				expectError: `failed to execute: {{1 ? 1 : 2}}: invalid operation: operator ? not defined on 1 (value of type int64)`,
			},
			`defined(v) ? v : "default" + true`: {
				// "default" + true should not be evaluated
				str: `{{defined(v) ? v : "default" + true}}`,
				data: map[string]interface{}{
					"v": "override",
				},
				expect: "override",
			},
			`defined(v) ? v + true : "default"`: {
				// v + true should not be evaluated
				str:    `{{defined(v) ? v : "default"}}`,
				expect: "default",
			},
		}
		runExecute(t, tests)
	})

	t.Run("complicated", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"*,/ have precedence over +,-": {
				str:    `{{1 + 2 * 3 / 4 - 5}}`,
				expect: int64(-3),
			},
			"paren expr": {
				str:    `{{(1 + 2) * 3 / (4 - 5)}}`,
				expect: int64(-9),
			},
			"condition with paren": {
				str:    `{{ 1 != 2 && !(false || 1 >= 2)}}`,
				expect: true,
			},
			"conditional operator": {
				str:    `{{1 + 1 <= 2 ? 3 * 3 : 4 / 4}}`,
				expect: int64(9),
			},
		}
		runExecute(t, tests)
	})
}

type executeTestCase struct {
	str         string
	data        interface{}
	expect      interface{}
	expectError string
}

func runExecute(t *testing.T, tests map[string]executeTestCase) {
	t.Helper()
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
