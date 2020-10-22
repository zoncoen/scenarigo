package assert

import (
	"testing"

	"github.com/zoncoen/query-go"
)

func TestAnd(t *testing.T) {
	q := query.New()
	if err := And()(q).Assert(""); err == nil {
		t.Fatal("empty assertion list should be an error")
	}

	tests := map[string]struct {
		assertions []Assertion
		ok         interface{}
		ng         interface{}
	}{
		"one": {
			assertions: []Assertion{
				Equal(q, "one"),
			},
			ok: "one",
			ng: "1",
		},
		"multi": {
			assertions: []Assertion{
				Equal(q, "one"),
				NotZero(q),
			},
			ok: "one",
			ng: "1",
		},
	}
	for _, test := range tests {
		test := test
		and := And(test.assertions...)(q)
		if err := and.Assert(test.ok); err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if err := and.Assert(test.ng); err == nil {
			t.Error("expect error but no error")
		}
	}
}

func TestOr(t *testing.T) {
	q := query.New()
	if err := Or()(q).Assert(""); err == nil {
		t.Fatal("empty assertion list should be an error")
	}

	tests := map[string]struct {
		assertions []Assertion
		ok         interface{}
		ng         interface{}
	}{
		"one": {
			assertions: []Assertion{
				Equal(q, "two"),
			},
			ok: "two",
			ng: "2",
		},
		"multi": {
			assertions: []Assertion{
				Equal(q, "one"),
				Equal(q, "two"),
			},
			ok: "two",
			ng: "2",
		},
	}
	for _, test := range tests {
		test := test
		or := Or(test.assertions...)(q)
		if err := or.Assert(test.ok); err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if err := or.Assert(test.ng); err == nil {
			t.Error("expect error but no error")
		}
	}
}
