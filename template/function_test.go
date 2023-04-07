package template

import (
	"math"
	"testing"

	"github.com/zoncoen/scenarigo/internal/testutil"
)

func TestTypeConversion(t *testing.T) {
	t.Run("int()", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"convert int to int": {
				str:    "{{int(1)}}",
				expect: int64(1),
			},
			"convert uint to int": {
				str: "{{int(v)}}",
				data: map[string]interface{}{
					"v": uint(1),
				},
				expect: int64(1),
			},
			"failed to convert uint to int (overflow)": {
				str: "{{int(v)}}",
				data: map[string]interface{}{
					"v": uint(math.MaxInt64) + 1,
				},
				expectError: `failed to execute: {{int(v)}}: 9223372036854775808 overflows int`,
			},
			"convert float to int": {
				str:    "{{int(1.9)}}",
				expect: int64(1),
			},
			"convert string to int": {
				str:    `{{int("1")}}`,
				expect: int64(1),
			},
			"failed to convert string to int (overflow)": {
				str:         `{{int("9223372036854775808")}}`,
				expectError: `failed to execute: {{int("9223372036854775808")}}: strconv.ParseInt: parsing "9223372036854775808": value out of range`,
			},
			"failed to convert string to int": {
				str:         `{{int("1.9")}}`,
				expectError: `failed to execute: {{int("1.9")}}: strconv.ParseInt: parsing "1.9": invalid syntax`,
			},
			"failed to convert struct to int": {
				str: `{{int(v)}}`,
				data: map[string]interface{}{
					"v": struct{}{},
				},
				expectError: `failed to execute: {{int(v)}}: can't convert struct {} to int`,
			},
			"convert *int to int": {
				str: "{{int(v)}}",
				data: map[string]interface{}{
					"v": testutil.ToPtr(1),
				},
				expect: int64(1),
			},
			"failed to convert (*int)(nil) to int": {
				str: `{{int(v)}}`,
				data: map[string]interface{}{
					"v": (*int)(nil),
				},
				expectError: `failed to execute: {{int(v)}}: can't convert (*int)(nil) to int`,
			},
			"failed to convert untyped nil to int": {
				str: `{{int(v)}}`,
				data: map[string]interface{}{
					"v": nil,
				},
				expectError: `failed to execute: {{int(v)}}: can't convert nil to int`,
			},
		}
		runExecute(t, tests)
	})

	t.Run("uint()", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"convert int to uint": {
				str:    "{{uint(1)}}",
				expect: uint64(1),
			},
			"failed to convert int to uint": {
				str:         "{{uint(-1)}}",
				expectError: "failed to execute: {{uint(-1)}}: can't convert -1 to uint",
			},
			"convert uint to uint": {
				str: "{{uint(v)}}",
				data: map[string]interface{}{
					"v": uint(1),
				},
				expect: uint64(1),
			},
			"convert float to uint": {
				str:    "{{uint(1.9)}}",
				expect: uint64(1),
			},
			"convert string to uint": {
				str:    `{{uint("1")}}`,
				expect: uint64(1),
			},
			"failed to convert string to uint": {
				str:         `{{uint("-1")}}`,
				expectError: `failed to execute: {{uint("-1")}}: strconv.ParseUint: parsing "-1": invalid syntax`,
			},
			"failed to convert struct to uint": {
				str: `{{uint(v)}}`,
				data: map[string]interface{}{
					"v": struct{}{},
				},
				expectError: `failed to execute: {{uint(v)}}: can't convert struct {} to uint`,
			},
			"convert *uint to uint": {
				str: "{{uint(v)}}",
				data: map[string]interface{}{
					"v": testutil.ToPtr(uint(1)),
				},
				expect: uint64(1),
			},
			"failed to convert (*uint)(nil) to uint": {
				str: `{{uint(v)}}`,
				data: map[string]interface{}{
					"v": (*uint)(nil),
				},
				expectError: `failed to execute: {{uint(v)}}: can't convert (*uint)(nil) to uint`,
			},
			"failed to convert untyped nil to uint": {
				str: `{{uint(v)}}`,
				data: map[string]interface{}{
					"v": nil,
				},
				expectError: `failed to execute: {{uint(v)}}: can't convert nil to uint`,
			},
		}
		runExecute(t, tests)
	})

	t.Run("float()", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"convert int to float": {
				str:    "{{float(1)}}",
				expect: 1.0,
			},
			"convert uint to float": {
				str: "{{float(v)}}",
				data: map[string]interface{}{
					"v": uint(1),
				},
				expect: 1.0,
			},
			"convert float to float": {
				str:    "{{float(1.9)}}",
				expect: 1.9,
			},
			"convert string to float": {
				str:    `{{float("1.9")}}`,
				expect: 1.9,
			},
			"failed to convert string to float": {
				str:         `{{float("a")}}`,
				expectError: `failed to execute: {{float("a")}}: strconv.ParseFloat: parsing "a": invalid syntax`,
			},
			"failed to convert struct to float": {
				str: `{{float(v)}}`,
				data: map[string]interface{}{
					"v": struct{}{},
				},
				expectError: `failed to execute: {{float(v)}}: can't convert struct {} to float`,
			},
			"convert *float to float": {
				str: "{{float(v)}}",
				data: map[string]interface{}{
					"v": testutil.ToPtr(1.9),
				},
				expect: 1.9,
			},
			"failed to convert (*float)(nil) to float": {
				str: `{{float(v)}}`,
				data: map[string]interface{}{
					"v": (*float64)(nil),
				},
				expectError: `failed to execute: {{float(v)}}: can't convert (*float64)(nil) to float`,
			},
			"failed to convert untyped nil to float": {
				str: `{{float(v)}}`,
				data: map[string]interface{}{
					"v": nil,
				},
				expectError: `failed to execute: {{float(v)}}: can't convert nil to float`,
			},
		}
		runExecute(t, tests)
	})

	t.Run("bool()", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"convert bool to bool": {
				str:    "{{bool(true)}}",
				expect: true,
			},
			"failed to convert int to bool": {
				str:         `{{bool(1)}}`,
				expectError: "failed to execute: {{bool(1)}}: can't convert int to bool",
			},
			"convert *bool to bool": {
				str: "{{bool(v)}}",
				data: map[string]interface{}{
					"v": testutil.ToPtr(true),
				},
				expect: true,
			},
			"failed to convert (*bool)(nil) to bool": {
				str: `{{bool(v)}}`,
				data: map[string]interface{}{
					"v": (*bool)(nil),
				},
				expectError: `failed to execute: {{bool(v)}}: can't convert (*bool)(nil) to bool`,
			},
			"failed to convert untyped nil to bool": {
				str: `{{bool(v)}}`,
				data: map[string]interface{}{
					"v": nil,
				},
				expectError: `failed to execute: {{bool(v)}}: can't convert nil to bool`,
			},
		}
		runExecute(t, tests)
	})

	t.Run("string()", func(t *testing.T) {
		tests := map[string]executeTestCase{
			"convert int to string": {
				str:    "{{string(1)}}",
				expect: "1",
			},
			"convert uint to string": {
				str: "{{string(v)}}",
				data: map[string]interface{}{
					"v": uint(1),
				},
				expect: "1",
			},
			"convert float to string": {
				str:    "{{string(1.2345)}}",
				expect: "1.2345",
			},
			"convert string to string": {
				str:    `{{string("test")}}`,
				expect: "test",
			},
			"failed to convert struct to string": {
				str: `{{string(v)}}`,
				data: map[string]interface{}{
					"v": struct{}{},
				},
				expectError: `failed to execute: {{string(v)}}: can't convert struct {} to string`,
			},
			"convert *string to string": {
				str: `{{string(v)}}`,
				data: map[string]interface{}{
					"v": testutil.ToPtr("test"),
				},
				expect: "test",
			},
			"failed to convert (*string)(nil) to string": {
				str: `{{string(v)}}`,
				data: map[string]interface{}{
					"v": (*string)(nil),
				},
				expectError: `failed to execute: {{string(v)}}: can't convert (*string)(nil) to string`,
			},
			"failed to convert untyped nil to string": {
				str: `{{string(v)}}`,
				data: map[string]interface{}{
					"v": nil,
				},
				expectError: `failed to execute: {{string(v)}}: can't convert nil to string`,
			},
		}
		runExecute(t, tests)
	})
}
