package scenarigo

import (
	"bytes"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zoncoen/scenarigo/assert"
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/reporter"
	"github.com/zoncoen/scenarigo/schema"
)

func TestRunScenario_PrintDebugInfo(t *testing.T) {
	tests := map[string]struct {
		scenario *schema.Scenario
		expect   string
	}{
		"success": {
			scenario: &schema.Scenario{
				Steps: []*schema.Step{
					{
						Request: schema.Request{
							Invoker: invoker(func(ctx *context.Context) (*context.Context, interface{}, error) {
								ctx = ctx.WithRequest(map[string]string{"type": "request"})
								ctx = ctx.WithResponse(map[string]string{"type": "response"})
								return ctx, nil, nil
							}),
						},
						Expect: schema.Expect{
							AssertionBuilder: builder(func(ctx *context.Context) (assert.Assertion, error) {
								return assert.AssertionFunc(func(_ interface{}) error { return nil }), nil
							}),
						},
					},
				},
			},
			expect: `
=== RUN   
--- PASS:  (0.00s)
`,
		},
		"no debug info": {
			scenario: &schema.Scenario{
				Steps: []*schema.Step{
					{
						Request: schema.Request{
							Invoker: invoker(func(ctx *context.Context) (*context.Context, interface{}, error) {
								return nil, nil, errors.New("failed")
							}),
						},
						Expect: schema.Expect{
							AssertionBuilder: builder(func(ctx *context.Context) (assert.Assertion, error) {
								return nil, nil
							}),
						},
					},
				},
			},
			expect: `
=== RUN   
--- FAIL:  (0.00s)
    failed
`,
		},
		"request": {
			scenario: &schema.Scenario{
				Steps: []*schema.Step{
					{
						Request: schema.Request{
							Invoker: invoker(func(ctx *context.Context) (*context.Context, interface{}, error) {
								ctx = ctx.WithRequest(map[string]string{"type": "request"})
								return ctx, nil, nil
							}),
						},
						Expect: schema.Expect{
							AssertionBuilder: builder(func(ctx *context.Context) (assert.Assertion, error) {
								return nil, errors.New("failed")
							}),
						},
					},
				},
			},
			expect: `
=== RUN   
--- FAIL:  (0.00s)
    failed
    request:
        type: request
`,
		},
		"request and response": {
			scenario: &schema.Scenario{
				Steps: []*schema.Step{
					{
						Request: schema.Request{
							Invoker: invoker(func(ctx *context.Context) (*context.Context, interface{}, error) {
								ctx = ctx.WithRequest(map[string]string{"type": "request"})
								ctx = ctx.WithResponse(map[string]string{"type": "response"})
								return ctx, nil, nil
							}),
						},
						Expect: schema.Expect{
							AssertionBuilder: builder(func(ctx *context.Context) (assert.Assertion, error) {
								return nil, errors.New("failed")
							}),
						},
					},
				},
			},
			expect: `
=== RUN   
--- FAIL:  (0.00s)
    failed
    request:
        type: request
    response:
        type: response
`,
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			var b bytes.Buffer
			reporter.Run(func(rptr reporter.Reporter) {
				runScenario(context.New(rptr), test.scenario)
			}, reporter.WithWriter(&b))
			if diff := cmp.Diff(test.expect, "\n"+b.String()); diff != "" {
				t.Errorf("differs (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunScenario_FailFast(t *testing.T) {
	var called bool
	scenario := &schema.Scenario{
		Steps: []*schema.Step{
			{
				Request: schema.Request{
					Invoker: invoker(func(ctx *context.Context) (*context.Context, interface{}, error) {
						return ctx, nil, nil
					}),
				},
				Expect: schema.Expect{
					AssertionBuilder: builder(func(ctx *context.Context) (assert.Assertion, error) {
						return nil, errors.New("failed")
					}),
				},
			},
			{
				Request: schema.Request{
					Invoker: invoker(func(ctx *context.Context) (*context.Context, interface{}, error) {
						called = true
						return ctx, nil, nil
					}),
				},
				Expect: schema.Expect{
					AssertionBuilder: builder(func(ctx *context.Context) (assert.Assertion, error) {
						return assert.AssertionFunc(func(_ interface{}) error { return nil }), nil
					}),
				},
			},
		},
	}
	reporter.Run(func(rptr reporter.Reporter) {
		runScenario(context.New(rptr), scenario)
	})
	if called {
		t.Fatal("following steps should be skipped if the previous step failed")
	}
}
