package plugin

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/sosedoff/gitkit"
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
	gomod := `module main

go 1.17
`

	b, err := os.ReadFile("testdata/go1.17.mod")
	if err != nil {
		t.Fatal(err)
	}
	gomodWithRequire := string(b)

	setupGitServer(t)

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
			"update go.mod (remove replace)": {
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

replace google.golang.org/grpc v1.37.1 => google.golang.org/grpc v1.40.0
`, goVersion),
				},
				expectPluginPath: "plugin.so",
				expectGoMod: map[string]string{
					"src/go.mod": gomodWithRequire,
				},
			},
			`src is a "go gettable" remote git repository`: {
				config: `
schemaVersion: config/v1
plugins:
  gen/plugin.so:
    src: 127.0.0.1/plugin.git
`,
				files:            map[string]string{},
				expectPluginPath: "gen/plugin.so",
				expectGoMod:      map[string]string{},
			},
			`src is a remote git repository with version`: {
				config: `
schemaVersion: config/v1
plugins:
  gen/plugin.so:
    src: 127.0.0.1/plugin.git@v1.0.0
`,
				files:            map[string]string{},
				expectPluginPath: "gen/plugin.so",
				expectGoMod:      map[string]string{},
			},
			`src is a remote git repository with latest version`: {
				config: `
schemaVersion: config/v1
plugins:
  gen/plugin.so:
    src: 127.0.0.1/plugin.git@latest
`,
				files:            map[string]string{},
				expectPluginPath: "gen/plugin.so",
				expectGoMod:      map[string]string{},
			},
			`should escape file path of remote repository`: {
				config: `
schemaVersion: config/v1
plugins:
  gen/plugin.so:
    src: 127.0.0.1/NeedEscape.git
`,
				files:            map[string]string{},
				expectPluginPath: "gen/plugin.so",
				expectGoMod:      map[string]string{},
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
			"invalid version": {
				config: `
schemaVersion: config/v1
plugins:
  plugin.so:
    src: 127.0.0.1/plugin.git@v2.0.0
`,
				expect: "unknown revision v2.0.0",
			},
			"can't build remote module": {
				config: `
schemaVersion: config/v1
plugins:
  plugin.so:
    src: 127.0.0.1/not-plugin.git
`,
				expect: "-buildmode=plugin requires exactly one main package",
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

func setupGitServer(t *testing.T) {
	t.Helper()

	// create git objects for test repositories
	tempDir := t.TempDir()
	envs := []string{
		fmt.Sprintf("GIT_CONFIG_GLOBAL=%s", filepath.Join(tempDir, ".gitconfig")),
	}
	ctx := context.Background()
	if err := executeWithEnvs(ctx, envs, tempDir, "git", "config", "--global", "user.name", "scenarigo-test"); err != nil {
		t.Fatalf("git config failed: %s", err)
	}
	if err := executeWithEnvs(ctx, envs, tempDir, "git", "config", "--global", "user.email", "scenarigo-test@example.com"); err != nil {
		t.Fatalf("git config failed: %s", err)
	}
	repoDir := filepath.Join("testdata", "repos")
	entries, err := os.ReadDir(repoDir)
	if err != nil {
		t.Fatalf("failed to read directory: %s", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		wd := filepath.Join(repoDir, e.Name())
		if err := executeWithEnvs(ctx, envs, wd, "git", "init"); err != nil {
			t.Fatalf("git init failed: %s", err)
		}
		if err := executeWithEnvs(ctx, envs, wd, "git", "add", "-A"); err != nil {
			t.Fatalf("git add failed: %s", err)
		}
		if err := executeWithEnvs(ctx, envs, wd, "git", "commit", "-m", "commit"); err != nil {
			t.Fatalf("git commit failed: %s", err)
		}
		if err := executeWithEnvs(ctx, envs, wd, "git", "tag", "v1.0.0"); err != nil {
			t.Fatalf("git commit failed: %s", err)
		}
		if err := os.Rename(
			filepath.Join(repoDir, e.Name(), ".git"),
			filepath.Join(tempDir, fmt.Sprintf("%s", e.Name())),
		); err != nil {
			t.Fatalf("failed to rename: %s", err)
		}
	}

	git := gitkit.NewSSH(gitkit.Config{
		Dir:    tempDir,
		KeyDir: filepath.Join(tempDir, "ssh"),
	})
	git.PublicKeyLookupFunc = func(_ string) (*gitkit.PublicKey, error) {
		return &gitkit.PublicKey{}, nil
	}
	if err := git.Listen(":0"); err != nil {
		t.Fatalf("failed to listen: %s", err)
	}
	go git.Serve()
	t.Cleanup(func() {
		git.Stop()
	})

	u, err := url.Parse(fmt.Sprintf("http://%s", git.Address()))
	if err != nil {
		t.Fatalf("failed to parse URL: %s", err)
	}
	setEnv(t, "GIT_SSH_COMMAND", fmt.Sprintf("ssh -p %s -i %s -oStrictHostKeyChecking=no -F /dev/null", u.Port(), filepath.Join(tempDir, "ssh", "gitkit.rsa")))
	setEnv(t, "GOPRIVATE", "127.0.0.1")
}

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	v, ok := os.LookupEnv(key)
	os.Setenv(key, value)
	t.Cleanup(func() {
		if ok {
			os.Setenv(key, v)
		} else {
			os.Unsetenv(key)
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
