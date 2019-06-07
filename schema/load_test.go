package schema

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLoadScenarios(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tests := map[string]struct {
			path   string
			expect []*Scenario
		}{
			"valid": {
				path: "testdata/valid.yaml",

				expect: []*Scenario{
					{
						Title:       "echo-service",
						Description: "check echo-service",
						Vars:        map[string]interface{}{"message": "hello"},
						Steps: []*Step{
							{
								Title:       "POST /say",
								Description: "check to respond same message",
								Vars:        nil,
								Protocol:    "http",
								Request: map[interface{}]interface{}{"method": "POST",
									"url":  "{{env.ECHO_SERVICE_ADDR}}/say",
									"body": map[interface{}]interface{}{"message": "{{vars.message}}"},
								},
								Expect: map[interface{}]interface{}{"code": "OK",
									"body": map[interface{}]interface{}{"message": "{{request.body}}"},
								},
							},
						},
						filepath: "testdata/valid.yaml",
					},
				},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				got, err := LoadScenarios(test.path)
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if diff := cmp.Diff(test.expect, got, cmp.AllowUnexported(Scenario{})); diff != "" {
					t.Errorf("scenario differs (-want +got):\n%s", diff)
				}
			})
		}
	})
	t.Run("failure", func(t *testing.T) {
		tests := map[string]struct {
			path string
		}{
			"not found": {
				path: "notfound.yaml",
			},
			"invalid": {
				path: "testdata/invalid.yaml",
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
