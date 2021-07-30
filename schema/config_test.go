package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLoadConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		wd, err := os.Getwd()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		got, err := LoadConfig("testdata/config/valid.yaml", false)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		colored := true
		expect := &Config{
			SchemaVersion: "config/v1",
			Scenarios: []string{
				"a.yaml",
				"b.yaml",
			},
			PluginDirectory: "plugins",
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
			Root: filepath.Join(wd, "testdata/config"),
		}
		if diff := cmp.Diff(expect, got); diff != "" {
			t.Errorf("differs (-want +got):\n%s", diff)
		}
	})
	t.Run("failure", func(t *testing.T) {
		tests := map[string]struct {
			path   string
			expect string
		}{
			"not found": {
				path:   "testdata/config/not-found.yaml",
				expect: "open testdata/config/not-found.yaml: no such file or directory",
			},
			"empty": {
				path:   "testdata/config/empty.yaml",
				expect: "schemaVersion not found",
			},
			"no version": {
				path:   "testdata/config/no-version.yaml",
				expect: "schemaVersion not found",
			},
			"unknown version": {
				path: "testdata/config/unknown-version.yaml",
				expect: `
>  1 | schemaVersion: config/unknown
                      ^
unknown version "config/unknown"`,
			},
			"invalid version": {
				path: "testdata/config/invalid-version.yaml",
				expect: `
   1 | schemaVersion:
>  2 |   - config
         ^
   3 |   - v1
invalid version: cannot unmarshal []interface {} into Go value of type string`,
			},
		}
		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				_, err := LoadConfig(test.path, false)
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
