package scenarigo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/zoncoen/scenarigo/assert"
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/protocol"
	"github.com/zoncoen/scenarigo/reporter"
)

type testProtocol struct {
	name               string
	requstUnmarshaller func(f func(interface{}) error) (protocol.Invoker, error)
	invoker            invoker
	expectUnmarshaller func(f func(interface{}) error) (protocol.AssertionBuilder, error)
	builder            builder
}

func (p *testProtocol) Name() string { return p.name }

func (p *testProtocol) UnmarshalRequest(f func(interface{}) error) (protocol.Invoker, error) {
	if p.requstUnmarshaller != nil {
		return p.requstUnmarshaller(f)
	}
	return p.invoker, nil
}

func (p *testProtocol) UnmarshalExpect(f func(interface{}) error) (protocol.AssertionBuilder, error) {
	if p.expectUnmarshaller != nil {
		return p.expectUnmarshaller(f)
	}
	return p.builder, nil
}

type invoker func(*context.Context) (*context.Context, interface{}, error)

func (f invoker) Invoke(ctx *context.Context) (*context.Context, interface{}, error) {
	return f(ctx)
}

type builder func(*context.Context) (assert.Assertion, error)

func (f builder) Build(ctx *context.Context) (assert.Assertion, error) {
	return f(ctx)
}

func TestRunner_Run(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tests := map[string]struct {
			scenario string
			invoker  func(*context.Context) (*context.Context, interface{}, error)
			builder  func(*context.Context) (assert.Assertion, error)
		}{
			"simple": {
				scenario: "testdata/scenarios/simple.yaml",
				invoker:  func(ctx *context.Context) (*context.Context, interface{}, error) { return ctx, nil, nil },
				builder: func(ctx *context.Context) (assert.Assertion, error) {
					return assert.AssertionFunc(func(_ interface{}) error { return nil }), nil
				},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				var invoked, built bool
				p := &testProtocol{
					name: "test",
					invoker: invoker(func(ctx *context.Context) (*context.Context, interface{}, error) {
						invoked = true
						return test.invoker(ctx)
					}),
					builder: builder(func(ctx *context.Context) (assert.Assertion, error) {
						built = true
						return test.builder(ctx)
					}),
				}
				protocol.Register(p)
				defer protocol.Unregister(p.Name())

				r, err := NewRunner(WithScenarios(test.scenario))
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}

				var b bytes.Buffer
				ok := reporter.Run(func(rptr reporter.Reporter) {
					r.Run(context.New(rptr))
				}, reporter.WithWriter(&b))
				if !ok {
					t.Fatalf("scenario failed:\n%s", b.String())
				}
				if !invoked {
					t.Error("did not invoke")
				}
				if !built {
					t.Error("did not build the assertion")
				}
			})
		}
	})
	t.Run("failure", func(t *testing.T) {
		tests := map[string]struct {
			scenario string
			invoker  func(*context.Context) (*context.Context, interface{}, error)
			builder  func(*context.Context) (assert.Assertion, error)
		}{
			"failed to invoke": {
				scenario: "testdata/scenarios/simple.yaml",
				invoker: func(ctx *context.Context) (*context.Context, interface{}, error) {
					return ctx, nil, errors.New("some error occurred")
				},
			},
			"failed to build the assertion": {
				scenario: "testdata/scenarios/simple.yaml",
				invoker:  func(ctx *context.Context) (*context.Context, interface{}, error) { return ctx, nil, nil },
				builder:  func(ctx *context.Context) (assert.Assertion, error) { return nil, errors.New("some error occurred") },
			},
			"assertion error": {
				scenario: "testdata/scenarios/simple.yaml",
				invoker:  func(ctx *context.Context) (*context.Context, interface{}, error) { return ctx, nil, nil },
				builder: func(ctx *context.Context) (assert.Assertion, error) {
					return assert.AssertionFunc(func(_ interface{}) error { return errors.New("some error occurred") }), nil
				},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				var invoked, built bool
				p := &testProtocol{
					name: "test",
					invoker: invoker(func(ctx *context.Context) (*context.Context, interface{}, error) {
						invoked = true
						return test.invoker(ctx)
					}),
					builder: builder(func(ctx *context.Context) (assert.Assertion, error) {
						built = true
						return test.builder(ctx)
					}),
				}
				protocol.Register(p)
				defer protocol.Unregister(p.Name())

				r, err := NewRunner(WithScenarios(test.scenario))
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}

				var b bytes.Buffer
				ok := reporter.Run(func(rptr reporter.Reporter) {
					r.Run(context.New(rptr))
				}, reporter.WithWriter(&b))
				if ok {
					t.Fatal("test passed")
				}
				if test.invoker != nil && !invoked {
					t.Error("did not invoke")
				}
				if test.builder != nil && !built {
					t.Error("did not build the assertion")
				}
			})
		}
	})
}

func TestRunner_Run_Scenarios(t *testing.T) {
	tests := map[string]struct {
		scenario string
		setup    func(*testing.T) func()
	}{
		"http": {
			scenario: "testdata/scenarios/http.yaml",
			setup: func(t *testing.T) func() {
				t.Helper()
				token := "XXXXX"
				mux := http.NewServeMux()
				mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				})
				mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
					auth := r.Header.Get("Authorization")
					if auth != fmt.Sprintf("Bearer %s", token) {
						w.WriteHeader(http.StatusForbidden)
						return
					}
					body := map[string]string{}
					d := json.NewDecoder(r.Body)
					defer r.Body.Close()
					if err := d.Decode(&body); err != nil {
						t.Fatalf("failed to decode request body: %s", err)
					}
					var msg string
					if m, ok := body["message"]; ok {
						msg = m
					}
					b, err := json.Marshal(map[string]string{
						"message": msg,
					})
					if err != nil {
						t.Fatalf("failed to marshal: %s", err)
					}
					w.Write(b)
				})

				s := httptest.NewServer(mux)
				if err := os.Setenv("TEST_ADDR", s.URL); err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if err := os.Setenv("TEST_TOKEN", token); err != nil {
					t.Fatalf("unexpected error: %s", err)
				}

				return func() {
					defer s.Close()
					defer os.Unsetenv("TEST_ADDR")
					defer os.Unsetenv("TEST_TOKEN")
				}
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			teardown := test.setup(t)
			defer teardown()

			r, err := NewRunner(WithScenarios(test.scenario))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			var b bytes.Buffer
			ok := reporter.Run(func(rptr reporter.Reporter) {
				r.Run(context.New(rptr))
			}, reporter.WithWriter(&b))
			if !ok {
				t.Fatalf("scenario failed:\n%s", b.String())
			}
		})
	}
}
