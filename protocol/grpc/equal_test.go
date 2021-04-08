package grpc

import (
	"testing"

	"github.com/zoncoen/scenarigo/testdata/gen/pb/test"
)

func TestEqualEnum(t *testing.T) {
	tests := map[string]struct {
		expected interface{}
		got      interface{}
		ok       bool
	}{
		"expected is not string": {
			expected: 1,
			got:      test.UserType_CUSTOMER,
		},
		"got is not enum": {
			expected: "CUSTOMER",
			got:      1,
		},
		"equals": {
			expected: "CUSTOMER",
			got:      test.UserType_CUSTOMER,
			ok:       true,
		},
		"not equals": {
			expected: "CUSTOMER",
			got:      test.UserType_STAFF,
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			ok, err := equalEnum(test.expected, test.got)
			if ok != test.ok {
				t.Errorf("expect %t but got %t", test.ok, ok)
			}
			if err != nil {
				t.Error(err)
			}
		})
	}
}
