package scenarigo

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	goplugin "plugin"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/plugin"
	"github.com/zoncoen/scenarigo/reporter"
	"github.com/zoncoen/scenarigo/schema"
)

func TestRunnerWithScenarios(t *testing.T) {
	scenariosPath := filepath.Join("test", "e2e", "testdata", "scenarios")
	runner, err := NewRunner(WithScenarios(scenariosPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(runner.scenarioFiles) == 0 {
		t.Fatal("failed to set scenario files")
	}
	for _, file := range runner.scenarioFiles {
		if !yamlPattern.MatchString(file) {
			t.Fatalf("invalid scenario file: %s", file)
		}
	}
}

func TestRunnerWithOptionsFromEnv(t *testing.T) {
	if err := os.Setenv(envScenarigoColor, "true"); err != nil {
		t.Fatalf("%+v", err)
	}
	defer os.Unsetenv(envScenarigoColor)
	runner, err := NewRunner(
		WithOptionsFromEnv(true),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !runner.enabledColor {
		t.Fatalf("failed to set enabledColor from env")
	}
}

func TestRunner(t *testing.T) {
	tests := map[string]struct {
		path  string
		yaml  string
		setup func(*context.Context) func(*context.Context)
	}{
		"run step with include": {
			path: filepath.Join("testdata", "use_include.yaml"),
			setup: func(ctx *context.Context) func(*context.Context) {
				mux := http.NewServeMux()
				mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
					defer r.Body.Close()
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.Copy(w, r.Body)
				})

				s := httptest.NewServer(mux)
				if err := os.Setenv("TEST_ADDR", s.URL); err != nil {
					ctx.Reporter().Fatalf("unexpected error: %s", err)
				}

				return func(*context.Context) {
					s.Close()
					os.Unsetenv("TEST_ADDR")
				}
			},
		},
		"run with yaml": {
			yaml: `
---
title: /echo
steps:
- title: POST /echo
  protocol: http
  request:
    method: POST
    url: "{{env.TEST_ADDR}}/echo"
    body:
      message: "hello"
  expect:
    code: 200
    body:
      message: "hello"
`,
			setup: func(ctx *context.Context) func(*context.Context) {
				mux := http.NewServeMux()
				mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
					defer r.Body.Close()
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.Copy(w, r.Body)
				})

				s := httptest.NewServer(mux)
				if err := os.Setenv("TEST_ADDR", s.URL); err != nil {
					ctx.Reporter().Fatalf("unexpected error: %s", err)
				}

				return func(*context.Context) {
					s.Close()
					os.Unsetenv("TEST_ADDR")
				}
			},
		},
	}
	for _, test := range tests {
		var opts []func(*Runner) error
		if test.path != "" {
			opts = append(opts, WithScenarios(test.path))
		}
		if test.yaml != "" {
			opts = append(opts, WithScenariosFromReader(strings.NewReader(test.yaml)))
		}
		runner, err := NewRunner(opts...)
		if err != nil {
			t.Fatal(err)
		}
		if test.setup != nil {
			runner.pluginSetups["setup"] = func(ctx *plugin.Context) (*plugin.Context, func(*plugin.Context)) {
				return ctx, test.setup(ctx)
			}
		}
		var b bytes.Buffer
		ok := reporter.Run(func(rptr reporter.Reporter) {
			runner.Run(context.New(rptr))
		}, reporter.WithWriter(&b))
		if !ok {
			t.Fatalf("scenario failed:\n%s", b.String())
		}
	}
}

func TestRunnerFail(t *testing.T) {
	tests := map[string]struct {
		path  string
		yaml  string
		setup func(*testing.T) func()
	}{
		"include invalid yaml": {
			path: filepath.Join("testdata", "use_include_error.yaml"),
			setup: func(t *testing.T) func() {
				t.Helper()

				mux := http.NewServeMux()
				mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					defer r.Body.Close()
					w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
					_, _ = io.Copy(w, r.Body)
				})

				s := httptest.NewServer(mux)
				if err := os.Setenv("TEST_ADDR", s.URL); err != nil {
					t.Fatalf("unexpected error: %s", err)
				}

				return func() {
					s.Close()
					os.Unsetenv("TEST_ADDR")
				}
			},
		},
		"run with yaml": {
			yaml:  `invalid: value`,
			setup: func(t *testing.T) func() { return func() {} },
		},
	}
	for _, test := range tests {
		teardown := test.setup(t)
		defer teardown()

		var opts []func(*Runner) error
		if test.path != "" {
			opts = append(opts, WithScenarios(test.path))
		}
		if test.yaml != "" {
			opts = append(opts, WithScenariosFromReader(strings.NewReader(test.yaml)))
		}
		runner, err := NewRunner(opts...)
		if err != nil {
			t.Fatal(err)
		}
		var b bytes.Buffer
		ok := reporter.Run(func(rptr reporter.Reporter) {
			runner.Run(context.New(rptr))
		}, reporter.WithWriter(&b))
		if ok {
			t.Fatal("expected error but no error")
		}
	}
}

func TestRunner_ScenarioFiles(t *testing.T) {
	scenariosPath := filepath.Join("test", "e2e", "testdata", "scenarios")
	runner, err := NewRunner(WithScenarios(scenariosPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(runner.ScenarioFiles()) == 0 {
		t.Fatal("failed to get scenario files")
	}
}

func TestWithConfig(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	pluginDir := "plugins"
	pluginDirAbs, err := filepath.Abs(pluginDir)
	if err != nil {
		t.Fatal(err)
	}
	pluginOpen = func(_ string) (lookupper, error) {
		return mapLookupper{
			"Setup":     setup,
			"SetupFunc": setupFunc,
		}, nil
	}
	colored := true
	tests := map[string]struct {
		config *schema.Config
		expect *Runner
	}{
		"nil": {
			config: nil,
			expect: &Runner{
				rootDir: wd,
			},
		},
		"empty": {
			config: &schema.Config{},
			expect: &Runner{
				scenarioFiles: []string{},
				rootDir:       wd,
			},
		},
		"root directory": {
			config: &schema.Config{
				Root: "/path/to/directory",
			},
			expect: &Runner{
				scenarioFiles: []string{},
				rootDir:       "/path/to/directory",
			},
		},
		"scenarios": {
			config: &schema.Config{
				Scenarios: []string{"testdata/use_include.yaml"},
			},
			expect: &Runner{
				scenarioFiles: []string{filepath.Join(wd, "testdata/use_include.yaml")},
				rootDir:       wd,
			},
		},
		"plugin directory": {
			config: &schema.Config{
				PluginDirectory: pluginDir,
			},
			expect: &Runner{
				scenarioFiles: []string{},
				pluginDir:     &pluginDirAbs,
				rootDir:       wd,
			},
		},
		"plugin setup (function)": {
			config: &schema.Config{
				Plugins: map[string]schema.PluginConfig{
					"simple.so": {
						Setup: "{{Setup}}",
					},
					"http.so": {}, // no setup function
				},
			},
			expect: &Runner{
				scenarioFiles: []string{},
				pluginSetups: map[string]plugin.SetupFunc{
					"simple.so": nil,
				},
				rootDir: wd,
			},
		},
		"plugin setup (variable)": {
			config: &schema.Config{
				Plugins: map[string]schema.PluginConfig{
					"simple.so": {
						Setup: "{{SetupFunc}}",
					},
				},
			},
			expect: &Runner{
				scenarioFiles: []string{},
				pluginSetups: map[string]plugin.SetupFunc{
					"simple.so": nil,
				},
				rootDir: wd,
			},
		},
		"output colored": {
			config: &schema.Config{
				Output: schema.OutputConfig{
					Colored: &colored,
				},
			},
			expect: &Runner{
				scenarioFiles: []string{},
				enabledColor:  colored,
				rootDir:       wd,
			},
		},
		"output report": {
			config: &schema.Config{
				Output: schema.OutputConfig{
					Report: schema.ReportConfig{
						JSON: schema.JSONReportConfig{
							Filename: "report.json",
						},
						JUnit: schema.JUnitReportConfig{
							Filename: "report.json",
						},
					},
				},
			},
			expect: &Runner{
				scenarioFiles: []string{},
				reportConfig: schema.ReportConfig{
					JSON: schema.JSONReportConfig{
						Filename: "report.json",
					},
					JUnit: schema.JUnitReportConfig{
						Filename: "report.json",
					},
				},
				rootDir: wd,
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := NewRunner(WithConfig(test.config))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(test.expect, got,
				cmp.AllowUnexported(Runner{}),
				cmp.FilterPath(func(p cmp.Path) bool {
					switch p.String() {
					case "pluginSetups", "pluginTeardowns":
						return true
					}
					return false
				}, cmp.Ignore()),
			); diff != "" {
				t.Errorf("differs (-want +got):\n%s", diff)
			}
			if g, e := len(got.pluginSetups), len(test.expect.pluginSetups); g != e {
				t.Errorf("expect %d setups but got %d", e, g)
			}
			if g, e := len(got.pluginTeardowns), len(test.expect.pluginTeardowns); g != e {
				t.Errorf("expect %d teardowns but got %d", e, g)
			}
		})
	}
}

type mapLookupper map[string]interface{}

func (m mapLookupper) Lookup(name string) (goplugin.Symbol, error) {
	if v, ok := m[name]; ok {
		return goplugin.Symbol(v), nil
	}
	return nil, fmt.Errorf("%q not found", name)
}

func setup(ctx *plugin.Context) (*plugin.Context, func(*plugin.Context)) {
	ctx.Reporter().Log("setup")
	return ctx, nil
}

var setupFunc = plugin.SetupFunc(setup)

func TestWriteTestReport(t *testing.T) {
	tests := map[string]struct {
		config schema.ReportConfig
		files  []string
	}{
		"default": {},
		"json": {
			config: schema.ReportConfig{
				JSON: schema.JSONReportConfig{
					Filename: "report.json",
				},
			},
			files: []string{"report.json"},
		},
		"junit": {
			config: schema.ReportConfig{
				JUnit: schema.JUnitReportConfig{
					Filename: "junit.xml",
				},
			},
			files: []string{"junit.xml"},
		},
		"all": {
			config: schema.ReportConfig{
				JSON: schema.JSONReportConfig{
					Filename: "report.json",
				},
				JUnit: schema.JUnitReportConfig{
					Filename: "junit.xml",
				},
			},
			files: []string{"report.json", "junit.xml"},
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			r, err := NewRunner(WithConfig(&schema.Config{
				Output: schema.OutputConfig{
					Report: test.config,
				},
				Root: dir,
			}))
			if err != nil {
				t.Fatalf("failed to create a runner: %s", err)
			}

			if success := reporter.Run(func(rptr reporter.Reporter) {
				r.writeTestReport(context.New(rptr))
			}); !success {
				t.Fatal("runner failed")
			}

			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatalf("failed to read directory: %s", err)
			}
			filenames := map[string]struct{}{}
			for _, e := range entries {
				filenames[e.Name()] = struct{}{}
			}
			for _, file := range test.files {
				if _, ok := filenames[file]; !ok {
					t.Errorf("%q not found", file)
				} else {
					delete(filenames, file)
				}
			}
			for name := range filenames {
				t.Errorf("unexpected file %q created", name)
			}
		})
	}
}
