package assert

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/zoncoen/query-go"
	"github.com/zoncoen/scenarigo/testdata/gen/pb/test"
)

func TestEqual(t *testing.T) {
	type myString string
	tests := map[string]struct {
		expected interface{}
		ok       interface{}
		ng       interface{}
	}{
		"integer": {
			expected: 1,
			ok:       1,
			ng:       2,
		},
		"integer (type conversion)": {
			expected: 1,
			ok:       uint64(1),
			ng:       uint64(2),
		},
		"string": {
			expected: "test",
			ok:       "test",
			ng:       "develop",
		},
		"string (type conversion)": {
			expected: "test",
			ok:       myString("test"),
			ng:       myString("develop"),
		},
		"enum integer": {
			expected: int(test.UserType_CUSTOMER),
			ok:       test.UserType_CUSTOMER,
			ng:       test.UserType_USER_TYPE_UNSPECIFIED,
		},
		"enum string": {
			expected: test.UserType_CUSTOMER.String(),
			ok:       test.UserType_CUSTOMER,
			ng:       test.UserType_USER_TYPE_UNSPECIFIED,
		},
	}
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			assertion := Equal(query.New(), tc.expected)
			if err := assertion.Assert(tc.ok); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
			if err := assertion.Assert(tc.ng); err == nil {
				t.Errorf("expected error but no error")
			}
		})
	}
}

func TestConvert(t *testing.T) {
	type myString string
	tests := []struct {
		expected interface{}
		got      interface{}
		ok       bool
	}{
		{
			expected: 5,
			got:      uint64(5),
			ok:       true,
		},
		{
			expected: "test",
			got:      5,
			ok:       false,
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			_, err := convert(test.expected, reflect.TypeOf(test.got))
			if test.ok && err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if !test.ok && err == nil {
				t.Fatal("expected error but no error")
			}
		})
	}
}
