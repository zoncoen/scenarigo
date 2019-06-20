package context

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zoncoen/query-go"
	"github.com/zoncoen/scenarigo/reporter"
)

func TestContext_ExtractKey(t *testing.T) {
	vars := map[string]string{
		"foo": "bar",
	}
	tests := map[string]struct {
		ctx    func(*Context) *Context
		query  string
		expect interface{}
	}{
		"vars": {
			ctx: func(ctx *Context) *Context {
				return ctx.WithVars(vars)
			},
			query:  "vars.foo",
			expect: "bar",
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			ctx := New(reporter.FromT(t))
			ctx = test.ctx(ctx)
			q, err := query.ParseString(test.query)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			got, err := q.Extract(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(test.expect, got); diff != "" {
				t.Errorf("differs: (-want +got)\n%s", diff)
			}
		})
	}
}
