package schema

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestLoadConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tests := map[string]struct {
			path           string
			expectComments yaml.CommentMap
		}{
			"without comment": {
				path: "testdata/config/valid.yaml",
			},
			"with comment": {
				path: "testdata/config/valid-with-comment.yaml",
				expectComments: yaml.CommentMap{
					"$.schemaVersion": []*yaml.Comment{
						{
							Texts:    []string{" comment1", " comment2"},
							Position: yaml.CommentHeadPosition,
						},
					},
					"$.plugins.'remote-with-version.so'.src": []*yaml.Comment{
						{
							Texts:    []string{" comment3"},
							Position: yaml.CommentLinePosition,
						},
					},
				},
			},
		}
		re := regexp.MustCompile(".ytt.yaml$")
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				wd, err := os.Getwd()
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				got, err := LoadConfig(test.path)
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				colored := true
				expect := &Config{
					SchemaVersion: "config/v1",
					Scenarios: []string{
						"scenarios/a.yaml",
						"scenarios/b.yaml",
					},
					PluginDirectory: "gen",
					Plugins: PluginConfigMap{
						"local.so": {
							Order: 1,
							Name:  "local.so",
							Src:   "./plugin",
						},
						"remote.so": {
							Order: 2,
							Name:  "remote.so",
							Src:   "github.com/zoncoen/scenarigo",
						},
						"remote-with-version.so": {
							Order: 3,
							Name:  "remote-with-version.so",
							Src:   "github.com/zoncoen/scenarigo@v1.0.0",
						},
					},
					Input: InputConfig{
						Excludes: []Regexp{
							{
								Regexp: re,
								str:    ".ytt.yaml$",
							},
						},
						YAML: YAMLInputConfig{
							YTT: YTTConfig{
								Enabled: true,
								DefaultFiles: []string{
									"default.yaml",
								},
							},
						},
					},
					Output: OutputConfig{
						Verbose: true,
						Colored: &colored,
						Report: ReportConfig{
							JSON: JSONReportConfig{
								Filename: "report.json",
							},
							JUnit: JUnitReportConfig{
								Filename: "junit.xml",
							},
						},
					},
					Root:     filepath.Join(wd, "testdata/config"),
					Comments: test.expectComments,
				}
				if diff := cmp.Diff(expect, got, cmp.AllowUnexported(Regexp{}), cmpopts.IgnoreUnexported(regexp.Regexp{})); diff != "" {
					t.Errorf("differs (-want +got):\n%s", diff)
				}

				b, err := yaml.MarshalWithOptions(got, yaml.WithComment(got.Comments))
				if err != nil {
					t.Fatalf("failed to marshal: %s", err)
				}
				eb, err := os.ReadFile(test.path)
				if err != nil {
					t.Fatalf("failed to read file: %s", err)
				}
				if got, expect := string(b), string(eb); got != expect {
					dmp := diffmatchpatch.New()
					diffs := dmp.DiffMain(expect, got, false)
					t.Errorf("differs:\n%s", dmp.DiffPrettyText(diffs))
				}
			})
		}
	})

	t.Run("failure", func(t *testing.T) {
		tests := map[string]struct {
			path   string
			expect string
		}{
			"empty": {
				path:   "testdata/config/empty.yaml",
				expect: "empty config",
			},
			"multi document": {
				path:   "testdata/config/multi.yaml",
				expect: "must be a config document but contains more than one document",
			},
			"no version": {
				path:   "testdata/config/no-version.yaml",
				expect: "schemaVersion not found",
			},
			"unknown version": {
				path: "testdata/config/unknown-version.yaml",
				expect: `unknown version "config/unknown"
    >  1 | schemaVersion: config/unknown
                          ^
`,
			},
			"invalid version": {
				path: "testdata/config/invalid-version.yaml",
				expect: `invalid version: [2:3] cannot unmarshal []interface {} into Go value of type string
       1 | schemaVersion:
    >  2 |   - config
             ^
       3 |   - v1`,
			},
			"invalid scenarios": {
				path: "testdata/config/invalid-scenarios.yaml",
				expect: `1 error occurred: scenarios/invalid.yaml: no such file or directory
       1 | schemaVersion: config/v1
       2 | scenarios:
    >  3 |   - scenarios/invalid.yaml
               ^
`,
			},
			"plugin src not found": {
				path: "testdata/config/invalid-plugin-src-not-found.yaml",
				expect: `1 error occurred: invalid: no such file or directory
       1 | schemaVersion: config/v1
       2 | plugins:
       3 |   foo.so:
    >  4 |     src: invalid
                    ^
`,
			},
		}
		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				_, err := LoadConfig(test.path)
				if err == nil {
					t.Fatal("no error")
				}
				if got := err.Error(); test.expect != got {
					t.Errorf("\n=== expect ===\n%s\n=== got ===\n%s\n", test.expect, got)
				}
			})
		}
	})
}

func TestPluginConfigMap_UnmarshalYAML(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tests := map[string]struct {
			in     string
			expect PluginConfigMap
		}{
			"empty": {
				expect: PluginConfigMap{},
			},
			"not empty": {
				in: `
1: {}
plugin:
  src: ./src
`,
				expect: PluginConfigMap{
					"1": {
						Order: 1,
						Name:  "1",
					},
					"plugin": {
						Order: 2,
						Name:  "plugin",
						Src:   "./src",
					},
				},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				var got PluginConfigMap
				if err := got.UnmarshalYAML([]byte(test.in)); err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if diff := cmp.Diff(test.expect, got); diff != "" {
					t.Errorf("differs (-want +got):\n%s", diff)
				}
			})
		}
	})
	t.Run("failure", func(t *testing.T) {
		tests := map[string]struct {
			in     string
			expect string
		}{
			"failed to marshal": {
				in: "test",
				expect: `[1:1] string was used where mapping is expected
>  1 | test
       ^
`,
			},
			"key is not a string": {
				in:     "1.1: {}",
				expect: "value of type float64 is not assignable to type string",
			},
			"value is not a mapping": {
				in:     "plugin: test",
				expect: "string was used where mapping is expected",
			},
			"src is not a string": {
				in: `
plugin:
  src: 1.1`,
				expect: "value of type float64 is not assignable to type string",
			},
			"unknown field": {
				in: `
plugin:
  test: dir`,
				expect: `unknown field "test"`,
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				var got PluginConfigMap
				if err := got.UnmarshalYAML([]byte(test.in)); err == nil {
					t.Fatal("no error")
				} else {
					if got, expect := err.Error(), test.expect; got != expect {
						dmp := diffmatchpatch.New()
						diffs := dmp.DiffMain(expect, got, false)
						t.Errorf("error differs:\n%s", dmp.DiffPrettyText(diffs))
					}
				}
			})
		}
	})
}

func TestToYAMLString(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tests := map[string]struct {
			in     interface{}
			expect string
		}{
			"string": {
				in:     "test",
				expect: "test",
			},
			"uint64": {
				in:     uint64(1),
				expect: "1",
			},
			"int64": {
				in:     int64(-1),
				expect: "-1",
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				got, err := toYAMLString(test.in)
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if expect := test.expect; got != expect {
					t.Fatalf("expect %s but got %s", expect, got)
				}
			})
		}
	})
	t.Run("failure", func(t *testing.T) {
		_, err := toYAMLString(1.1)
		if err == nil {
			t.Fatal("no error")
		}
		if got, expect := err.Error(), "value of type float64 is not assignable to type string"; got != expect {
			t.Fatalf("expect %q but got %q", expect, got)
		}
	})
}

func TestPluginConfigMap_MarshalYAML(t *testing.T) {
	v := PluginConfigMap{
		"plugin.so": PluginConfig{
			Order: 2,
			Name:  "plugin.so",
			Src:   "src",
		},
		"-1": PluginConfig{
			Order: 1,
			Name:  "-1",
			Src:   "3",
		},
	}
	b, err := yaml.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal: %s", err)
	}
	expect := `"-1":
  src: "3"
plugin.so:
  src: src
`
	if got := string(b); got != expect {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(expect, got, false)
		t.Errorf("differs:\n%s", dmp.DiffPrettyText(diffs))
	}
}

func TestPluginConfigMap_ToSlice(t *testing.T) {
	v := PluginConfigMap{}
	if err := yaml.UnmarshalWithOptions([]byte(`
-1:
  src: 3
  test:dada
	`), &v, yaml.Strict()); err != nil {
		t.Fatal(err)
	}
	m := PluginConfigMap(map[string]PluginConfig{
		"plugin3.so": {
			Order: 3,
			Name:  "plugin3.so",
			Src:   "src3",
		},
		"plugin2.so": {
			Order: 2,
			Name:  "plugin2.so",
			Src:   "src2",
		},
		"plugin1.so": {
			Order: 1,
			Name:  "plugin1.so",
			Src:   "src1",
		},
	})
	if diff := cmp.Diff([]PluginConfig{
		{
			Order: 1,
			Name:  "plugin1.so",
			Src:   "src1",
		},
		{
			Order: 2,
			Name:  "plugin2.so",
			Src:   "src2",
		},
		{
			Order: 3,
			Name:  "plugin3.so",
			Src:   "src3",
		},
	}, m.ToSlice()); diff != "" {
		t.Errorf("differs (-want +got):\n%s", diff)
	}
}
