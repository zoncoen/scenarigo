package assert

import (
	"testing"
)

func TestAnd(t *testing.T) {
	if err := And().Assert(""); err == nil {
		t.Fatal("empty assertion list should be an error")
	}

	tests := map[string]struct {
		assertions []Assertion
		ok         interface{}
		ng         interface{}
	}{
		"one": {
			assertions: []Assertion{
				Equal("one"),
			},
			ok: "one",
			ng: "1",
		},
		"multi": {
			assertions: []Assertion{
				Equal("one"),
				NotZero(),
			},
			ok: "one",
			ng: "1",
		},
	}
	for _, test := range tests {
		test := test
		and := And(test.assertions...)
		if err := and.Assert(test.ok); err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if err := and.Assert(test.ng); err == nil {
			t.Error("expect error but no error")
		}
	}
}

func TestOr(t *testing.T) {
	if err := Or().Assert(""); err == nil {
		t.Fatal("empty assertion list should be an error")
	}

	tests := map[string]struct {
		assertions []Assertion
		ok         interface{}
		ng         interface{}
	}{
		"one": {
			assertions: []Assertion{
				Equal("two"),
			},
			ok: "two",
			ng: "2",
		},
		"multi": {
			assertions: []Assertion{
				Equal("one"),
				Equal("two"),
			},
			ok: "two",
			ng: "2",
		},
	}
	for _, test := range tests {
		test := test
		or := Or(test.assertions...)
		if err := or.Assert(test.ok); err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if err := or.Assert(test.ng); err == nil {
			t.Error("expect error but no error")
		}
	}
}
