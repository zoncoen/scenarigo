package plugin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/cobra"

	"github.com/zoncoen/scenarigo"
	"github.com/zoncoen/scenarigo/cmd/scenarigo/cmd/config"
)

var (
	bash string
	echo string
)

func init() {
	var err error
	bash, err = exec.LookPath("bash")
	if err != nil {
		panic("bash command not found")
	}
	echo, err = exec.LookPath("echo")
	if err != nil {
		panic("echo command not found")
	}
}

func TestBuild(t *testing.T) {
	// replace go.mod content for testing
	testGoMod, err := os.ReadFile("testdata/scenarigo.mod")
	if err != nil {
		t.Fatal(err)
	}
	gomodBytes := scenarigo.GoModBytes
	scenarigo.GoModBytes = testGoMod
	t.Cleanup(func() {
		scenarigo.GoModBytes = gomodBytes
	})

	goVersion := strings.TrimPrefix(runtime.Version(), "go")
	if parts := strings.Split(goVersion, "."); len(parts) > 2 {
		goVersion = fmt.Sprintf("%s.%s", parts[0], parts[1])
	}
	pluginCode := `package main

func Greet() string {
	return "Hello, world!"
}
`
	gomod := fmt.Sprintf(`module main

go %s
`, goVersion)

	b, err := os.ReadFile(fmt.Sprintf("testdata/go%s.mod", goVersion))
	if err != nil {
		t.Fatal(err)
	}
	gomodWithRequire := string(b)

	t.Run("sucess", func(t *testing.T) {
		tests := map[string]struct {
			config           string
			files            map[string]string
			expectPluginPath string
			expectGoMod      map[string]string
		}{
			"no plugins": {
				config: `
schemaVersion: config/v1
`,
			},
			"src is a file": {
				config: `
schemaVersion: config/v1
plugins:
  plugin.so:
    src: src/main.go
`,
				files: map[string]string{
					"src/main.go": pluginCode,
				},
				expectPluginPath: "plugin.so",
				expectGoMod: map[string]string{
					"src/go.mod": gomod,
				},
			},
			"src is a directory": {
				config: `
schemaVersion: config/v1
plugins:
  gen/plugin.so:
    src: src
`,
				files: map[string]string{
					"src/main.go": pluginCode,
				},
				expectPluginPath: "gen/plugin.so",
				expectGoMod: map[string]string{
					"src/go.mod": gomod,
				},
			},
			"specify pluginDirectory": {
				config: `
schemaVersion: config/v1
pluginDirectory: gen
plugins:
  plugin.so:
    src: src
`,
				files: map[string]string{
					"src/main.go": pluginCode,
				},
				expectPluginPath: "gen/plugin.so",
				expectGoMod: map[string]string{
					"src/go.mod": gomod,
				},
			},
			"update go.mod": {
				config: `
schemaVersion: config/v1
plugins:
  plugin.so:
    src: src/main.go
`,
				files: map[string]string{
					"src/main.go": `package main

import (
	_ "google.golang.org/grpc"
)

func Greet() string {
	return "Hello, world!"
}
`,
					"src/go.mod": fmt.Sprintf(`module main

go %s

require google.golang.org/grpc v1.37.1
`, goVersion),
				},
				expectPluginPath: "plugin.so",
				expectGoMod: map[string]string{
					"src/go.mod": gomodWithRequire,
				},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, config.DefaultConfigFileName)
				create(t, configPath, test.config)
				for p, content := range test.files {
					create(t, filepath.Join(tmpDir, p), content)
				}
				cmd := &cobra.Command{}
				cmd.Context()
				config.ConfigPath = configPath
				if err := build(cmd, []string{}); err != nil {
					t.Fatal(err)
				}
				if test.expectPluginPath != "" {
					if _, err := os.Stat(filepath.Join(tmpDir, test.expectPluginPath)); err != nil {
						t.Fatalf("plugin not found: %s", err)
					}
				}
				for path, expect := range test.expectGoMod {
					b, err := os.ReadFile(filepath.Join(tmpDir, path))
					if err != nil {
						t.Fatalf("failed read go.mod: %s", err)
					}
					if got := string(b); got != expect {
						dmp := diffmatchpatch.New()
						diffs := dmp.DiffMain(expect, got, false)
						t.Errorf("go.mod differs:\n%s", dmp.DiffPrettyText(diffs))
						fmt.Println(got)
					}
				}
			})
		}
	})
	t.Run("failure", func(t *testing.T) {
		tests := map[string]struct {
			config string
			files  map[string]string
			expect string
		}{
			"no config": {
				config: "",
				expect: "config file not found",
			},
			"specify invalid config": {
				config: "schemaVersion: test",
				expect: "failed to load config",
			},
			"src not found": {
				config: `
schemaVersion: config/v1
plugins:
  plugin.so:
    src: src/invalid.go
`,
				files: map[string]string{
					"src/main.go": pluginCode,
				},
				expect: "failed to find plugin src",
			},
			"build failed": {
				config: `
schemaVersion: config/v1
plugins:
  plugin.so:
    src: src/main.go
`,
				files: map[string]string{
					"src/main.go": `packag plugin`,
				},
				expect: "expected 'package', found packag",
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				tmpDir := t.TempDir()
				var configPath string
				if test.config != "" {
					configPath = filepath.Join(tmpDir, config.DefaultConfigFileName)
					create(t, configPath, test.config)
				}
				for p, content := range test.files {
					create(t, filepath.Join(tmpDir, p), content)
				}
				cmd := &cobra.Command{}
				config.ConfigPath = configPath
				err := build(cmd, []string{})
				if err == nil {
					t.Fatal("no error")
				}
				if !strings.Contains(err.Error(), test.expect) {
					t.Fatalf("unexpected error: %s", err)
				}
			})
		}
	})
}

func TestFindGoCmd(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		goVersion := runtime.Version()
		tests := map[string]struct {
			cmds   map[string]string
			expect string
		}{
			"found go command": {
				cmds: map[string]string{
					"go": fmt.Sprintf("go version %s linux/amd64", goVersion),
				},
				expect: "go",
			},
			fmt.Sprintf("found %s command", goVersion): {
				cmds: map[string]string{
					goVersion: fmt.Sprintf("go version %s linux/amd64", goVersion),
				},
				expect: goVersion,
			},
			fmt.Sprintf("different go version but %s found", goVersion): {
				cmds: map[string]string{
					"go":      "go version go1.15 linux/amd64",
					goVersion: fmt.Sprintf("go version %s linux/amd64", goVersion),
				},
				expect: goVersion,
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				tmpDir := t.TempDir()
				for cmd, stdout := range test.cmds {
					createExecutable(t, filepath.Join(tmpDir, cmd), stdout)
				}
				path := os.Getenv("PATH")
				t.Cleanup(func() {
					os.Setenv("PATH", path)
				})
				os.Setenv("PATH", tmpDir)
				goCmd, err := findGoCmd(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				if got, expect := filepath.Base(goCmd), test.expect; got != expect {
					t.Errorf("expect %q but got %q", expect, got)
				}
			})
		}
	})
	t.Run("failure", func(t *testing.T) {
		tests := map[string]struct {
			cmds   map[string]string
			expect string
		}{
			"command not found": {
				expect: "go command required",
			},
			"different go version": {
				cmds: map[string]string{
					"go": "go version go1.15 linux/amd64",
				},
				expect: "installed go1.15",
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				tmpDir := t.TempDir()
				for cmd, stdout := range test.cmds {
					createExecutable(t, filepath.Join(tmpDir, cmd), stdout)
				}
				path := os.Getenv("PATH")
				t.Cleanup(func() {
					os.Setenv("PATH", path)
				})
				os.Setenv("PATH", tmpDir)
				_, err := findGoCmd(context.Background())
				if err == nil {
					t.Fatal("no error")
				}
				if !strings.Contains(err.Error(), test.expect) {
					t.Fatalf("unexpected error: %s", err)
				}
			})
		}
	})
}

func create(t *testing.T, path, content string) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o777); err != nil {
		t.Fatalf("failed to create %s: %s", dir, err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create %s: %s", path, err)
	}
	defer f.Close()
	if _, err := f.Write([]byte(content)); err != nil {
		t.Fatalf("failed to write %s: %s", path, err)
	}
}

func createExecutable(t *testing.T, path, stdout string) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o777); err != nil {
		t.Fatalf("failed to create %s: %s", dir, err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o777)
	if err != nil {
		t.Fatalf("failed to create %s: %s", path, err)
	}
	defer f.Close()
	if _, err := f.Write([]byte(fmt.Sprintf("#!%s\n%s %q", bash, echo, stdout))); err != nil {
		t.Fatalf("failed to write %s: %s", path, err)
	}
}
