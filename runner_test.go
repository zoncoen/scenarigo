package scenarigo

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/zoncoen/scenarigo/context"
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
		setup func(*testing.T) func()
	}{
		"run step with include": {
			path: filepath.Join("testdata", "use_include.yaml"),
			setup: func(t *testing.T) func() {
				t.Helper()

				mux := http.NewServeMux()
				mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
					defer r.Body.Close()
					w.Header().Set("Content-Type", "application/json")
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
			setup: func(t *testing.T) func() {
				t.Helper()

				mux := http.NewServeMux()
				mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
					defer r.Body.Close()
					w.Header().Set("Content-Type", "application/json")
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

func TestRunner_ScenarioMap(t *testing.T) {
	runner, err := NewRunner(WithScenarios("testdata"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range runner.ScenarioFiles() {
		m, err := runner.ScenarioMap(context.FromT(t), file)
		if err != nil {
			t.Fatal(err)
		}
		if len(m) == 0 {
			t.Fatal("failed to get scenarios")
		}
		for _, steps := range m {
			if len(steps) == 0 {
				t.Fatal("failed to get steps from scenario map")
			}
		}
	}
}

func TestWithConfig(t *testing.T) {
	pluginDir := "plugins"
	pluginDirAbs, err := filepath.Abs(pluginDir)
	if err != nil {
		t.Fatal(err)
	}
	colored := true
	tests := map[string]struct {
		config *schema.Config
		expect *Runner
	}{
		"nil": {
			config: nil,
			expect: &Runner{},
		},
		"empty": {
			config: &schema.Config{},
			expect: &Runner{
				scenarioFiles: []string{},
			},
		},
		"scenarios": {
			config: &schema.Config{
				Scenarios: []string{"testdata/use_include.yaml"},
			},
			expect: &Runner{
				scenarioFiles: []string{"testdata/use_include.yaml"},
			},
		},
		"plugin directory": {
			config: &schema.Config{
				PluginDirectory: pluginDir,
			},
			expect: &Runner{
				scenarioFiles: []string{},
				pluginDir:     &pluginDirAbs,
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
			); diff != "" {
				t.Errorf("differs (-want +got):\n%s", diff)
			}
		})
	}
}

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
