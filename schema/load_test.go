package schema

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/zoncoen/scenarigo/assert"
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/protocol"
)

type testProtocol struct {
	name string
}

func (p *testProtocol) Name() string { return p.name }

func (p *testProtocol) UnmarshalRequest(b []byte) (protocol.Invoker, error) {
	var r request
	if err := yaml.Unmarshal(b, &r); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

func (p *testProtocol) UnmarshalExpect(b []byte) (protocol.AssertionBuilder, error) {
	var e expect
	if err := yaml.NewDecoder(bytes.NewBuffer(b), yaml.UseOrderedMap()).Decode(&e); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

type request map[interface{}]interface{}

func (r request) Invoke(ctx *context.Context) (*context.Context, interface{}, error) {
	return ctx, nil, nil
}

type expect map[interface{}]interface{}

func (e expect) Build(_ *context.Context) (assert.Assertion, error) {
	return assert.Build(e), nil
}

func TestLoadScenarios(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tests := map[string]struct {
			path             string
			scenarios        []*Scenario
			assertionBuilder interface{}
		}{
			"valid": {
				path: "testdata/valid.yaml",

				scenarios: []*Scenario{
					{
						Title:       "echo-service",
						Description: "check echo-service",
						Vars:        map[string]interface{}{"message": "hello"},
						Steps: []*Step{
							{
								Title:       "POST /say",
								Description: "check to respond same message",
								Vars:        nil,
								Protocol:    "test",
								Request: &request{
									"body": map[string]interface{}{
										"message": "{{vars.message}}",
									},
								},
								Expect: &expect{
									"body": yaml.MapSlice{
										yaml.MapItem{
											Key:   "message",
											Value: "{{request.body}}",
										},
									},
								},
							},
						},
						filepath: "testdata/valid.yaml",
					},
				},
			},
			"anchor": {
				path: "testdata/valid-anchor.yaml",

				scenarios: []*Scenario{
					{
						Title:       "echo-service",
						Description: "check echo-service",
						Vars:        map[string]interface{}{"message": "hello"},
						Steps: []*Step{
							{
								Title:       "POST /say",
								Description: "check to respond same message",
								Vars:        nil,
								Protocol:    "test",
								Request: &request{
									"body": map[string]interface{}{
										"message": "{{vars.message}}",
									},
								},
								Expect: &expect{
									"body": yaml.MapSlice{
										yaml.MapItem{
											Key:   "message",
											Value: "{{request.body}}",
										},
									},
								},
							},
						},
						filepath: "testdata/valid-anchor.yaml",
					},
				},
			},
			"without protocol": {
				path: "testdata/valid-without-protocol.yaml",

				scenarios: []*Scenario{
					{
						Title:       "echo-service",
						Description: "check echo-service",
						Vars:        map[string]interface{}{"message": "hello"},
						Steps: []*Step{
							{
								Include: "./valid.yaml",
							},
						},
						filepath: "testdata/valid-without-protocol.yaml",
					},
				},
			},
			"without expect": {
				path: "testdata/valid-without-expect.yaml",

				scenarios: []*Scenario{
					{
						Title:       "echo-service",
						Description: "check echo-service",
						Vars:        map[string]interface{}{"message": "hello"},
						Steps: []*Step{
							{
								Title:       "POST /say",
								Description: "check to respond same message",
								Vars:        nil,
								Protocol:    "test",
								Request: &request{
									"body": map[string]interface{}{
										"message": "{{vars.message}}",
									},
								},
							},
						},
						filepath: "testdata/valid-without-expect.yaml",
					},
				},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				p := &testProtocol{
					name: "test",
				}
				protocol.Register(p)
				defer protocol.Unregister(p.Name())

				got, err := LoadScenarios(test.path)
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if diff := cmp.Diff(test.scenarios, got,
					cmp.AllowUnexported(
						Scenario{},
					),
					cmp.FilterPath(func(path cmp.Path) bool {
						s := path.String()
						return s == "Node"
					}, cmp.Ignore()),
				); diff != "" {
					t.Errorf("scenario differs (-want +got):\n%s", diff)
				}
				for i, scn := range got {
					if g, e := scn.filepath, test.path; g != e {
						t.Errorf("[%d] expect %q but got %q", i, e, g)
					}
					if scn.Node == nil {
						t.Errorf("[%d] Node is nil", i)
					}
				}
			})
		}
	})
	t.Run("failure", func(t *testing.T) {
		p := &testProtocol{
			name: "test",
		}
		protocol.Register(p)
		defer protocol.Unregister(p.Name())

		tests := map[string]struct {
			path string
		}{
			"not found": {
				path: "notfound.yaml",
			},
			"parse error": {
				path: "testdata/parse-error.yaml",
			},
			"invalid": {
				path: "testdata/invalid.yaml",
			},
			"unknown protocol": {
				path: "testdata/unknown-protocol.yaml",
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				_, err := LoadScenarios(test.path)
				if err == nil {
					t.Fatal("expected error but no error")
				}
			})
		}
	})
}

func TestLoadScenariosFromReader(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tests := map[string]struct {
			yaml      string
			scenarios []*Scenario
		}{
			"valid": {
				yaml: `
title: echo-service
description: check echo-service
vars:
  message: hello
steps:
  - title: POST /say
    description: check to respond same message
    protocol: test
    request:
      body:
        message: "{{vars.message}}"
    expect:
      body:
        message: "{{request.body}}"
`,
				scenarios: []*Scenario{
					{
						Title:       "echo-service",
						Description: "check echo-service",
						Vars:        map[string]interface{}{"message": "hello"},
						Steps: []*Step{
							{
								Title:       "POST /say",
								Description: "check to respond same message",
								Vars:        nil,
								Protocol:    "test",
								Request: &request{
									"body": map[string]interface{}{
										"message": "{{vars.message}}",
									},
								},
								Expect: &expect{
									"body": yaml.MapSlice{
										yaml.MapItem{
											Key:   "message",
											Value: "{{request.body}}",
										},
									},
								},
							},
						},
					},
				},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				p := &testProtocol{
					name: "test",
				}
				protocol.Register(p)
				defer protocol.Unregister(p.Name())

				got, err := LoadScenariosFromReader(strings.NewReader(test.yaml))
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if diff := cmp.Diff(test.scenarios, got,
					cmp.AllowUnexported(
						Scenario{},
					),
					cmp.FilterPath(func(path cmp.Path) bool {
						s := path.String()
						return s == "Node"
					}, cmp.Ignore()),
				); diff != "" {
					t.Errorf("scenario differs (-want +got):\n%s", diff)
				}
				for i, scn := range got {
					if g, e := scn.filepath, ""; g != e {
						t.Errorf("[%d] expect %q but got %q", i, e, g)
					}
					if scn.Node == nil {
						t.Errorf("[%d] Node is nil", i)
					}
				}
			})
		}
	})
	t.Run("failure", func(t *testing.T) {
		tests := map[string]struct {
			r io.Reader
		}{
			"failed to read": {
				r: errReader{errors.New("read error")},
			},
			"parse error": {
				r: strings.NewReader(`
a:
- b
  c: d`),
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				_, err := LoadScenariosFromReader(test.r)
				if err == nil {
					t.Fatal("expected error but no error")
				}
			})
		}
	})
}

type errReader struct {
	err error
}

func (r errReader) Read(_ []byte) (int, error) { return 0, r.err }

func TestMarshalYAML(t *testing.T) {
	filename := "testdata/valid.yaml"

	p := &testProtocol{
		name: "test",
	}
	protocol.Register(p)
	defer protocol.Unregister(p.Name())

	scenarios, err := LoadScenarios(filename)
	if err != nil {
		t.Fatalf("failed to load scenarios: %s", err)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	for _, s := range scenarios {
		if err := enc.Encode(s); err != nil {
			t.Fatalf("failed to marshal to YAML: %s", err)
		}
	}

	b, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read file: %s", err)
	}

	if got, expect := buf.String(), string(b); got != expect {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(expect, got, false)
		t.Errorf("differs:\n%s", dmp.DiffPrettyText(diffs))
	}
}
