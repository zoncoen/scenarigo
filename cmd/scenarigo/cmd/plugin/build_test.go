package plugin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/sosedoff/gitkit"
	"github.com/spf13/cobra"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"

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
	goVersion := strings.TrimPrefix(runtime.Version(), "go")
	if parts := strings.Split(goVersion, "."); len(parts) > 2 {
		goVersion = fmt.Sprintf("%s.%s", parts[0], parts[1])
	}
	pluginCode := `package main

func Greet() string {
	return "Hello, world!"
}
`
	gomod := func(m string) string {
		return fmt.Sprintf(`module %s

go 1.17
`, m)
	}

	b, err := os.ReadFile("testdata/go1.17.mod.golden")
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
			skipOpen         bool
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
					"src/go.mod": gomod("plugins/plugin"),
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
					"src/go.mod": gomod("plugins/gen/plugin"),
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
					"src/go.mod": gomod("plugins/plugin"),
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
					"src/go.mod": fmt.Sprintf(`module plugins/plugin

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
					"src/go.mod": fmt.Sprintf(`module plugins/plugin

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
				skipOpen:         true,
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
				skipOpen:         true,
			},
			`src is a sub direcotry of remote git repository`: {
				config: `
schemaVersion: config/v1
plugins:
  gen/plugin.so:
    src: 127.0.0.1/sub.git/plugin@v1.0.0
`,
				files:            map[string]string{},
				expectPluginPath: "gen/plugin.so",
				expectGoMod:      map[string]string{},
			},
			`should escape file path of remote repository`: {
				config: `
schemaVersion: config/v1
plugins:
  gen/NeedEscape.so:
    src: 127.0.0.1/NeedEscape.git
`,
				files:            map[string]string{},
				expectPluginPath: "gen/NeedEscape.so",
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
				config.ConfigPath = configPath
				if err := build(cmd, []string{}); err != nil {
					t.Fatal(err)
				}
				if test.expectPluginPath != "" {
					if _, err := os.Stat(filepath.Join(tmpDir, test.expectPluginPath)); err != nil {
						t.Fatalf("plugin not found: %s", err)
					}
					if !test.skipOpen {
						openPlugin(t, filepath.Join(tmpDir, test.expectPluginPath))
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

func TestUpdateGoMod(t *testing.T) {
	tests := map[string]struct {
		gomod        string
		src          string
		requires     []*modfile.Require
		expect       string
		expectStdout string
	}{
		"do nothing": {
			gomod: `module plugin_module

go 1.17
`,
			expect: `module plugin_module

go 1.17
`,
		},
		"do nothing (no requires)": {
			gomod: `module plugin_module

go 1.17

require google.golang.org/grpc v1.37.1

require (
	github.com/golang/protobuf v1.4.2 // indirect
	golang.org/x/net v0.0.0-20190311183353-d8887717615a // indirect
	golang.org/x/sys v0.0.0-20190215142949-d0b11bdaac8a // indirect
	golang.org/x/text v0.3.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
)
`,
			src: `package main

import (
	_ "google.golang.org/grpc"
)
`,
			expect: `module plugin_module

go 1.17

require google.golang.org/grpc v1.37.1

require (
	github.com/golang/protobuf v1.4.2 // indirect
	golang.org/x/net v0.0.0-20190311183353-d8887717615a // indirect
	golang.org/x/sys v0.0.0-20190215142949-d0b11bdaac8a // indirect
	golang.org/x/text v0.3.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
)
`,
		},
		"do nothing (not used)": {
			gomod: `module plugin_module

go 1.17
`,
			requires: []*modfile.Require{
				{
					Mod: module.Version{
						Path:    "google.golang.org/grpc",
						Version: "v1.37.1",
					},
				},
			},
			expect: `module plugin_module

go 1.17
`,
		},
		"add require": {
			gomod: `module plugin_module

go 1.17
`,
			src: `package main

import (
	_ "google.golang.org/grpc"
)
`,
			requires: []*modfile.Require{
				{
					Mod: module.Version{
						Path:    "google.golang.org/grpc",
						Version: "v1.37.1",
					},
				},
			},
			expect: `module plugin_module

go 1.17

require google.golang.org/grpc v1.37.1

require (
	github.com/golang/protobuf v1.4.2 // indirect
	golang.org/x/net v0.0.0-20190311183353-d8887717615a // indirect
	golang.org/x/sys v0.0.0-20190215142949-d0b11bdaac8a // indirect
	golang.org/x/text v0.3.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
)
`,
		},
		"overwrite require": {
			gomod: `module plugin_module

go 1.17

require google.golang.org/grpc v1.37.1

require (
	github.com/golang/protobuf v1.4.2 // indirect
	golang.org/x/net v0.0.0-20190311183353-d8887717615a // indirect
	golang.org/x/sys v0.0.0-20190215142949-d0b11bdaac8a // indirect
	golang.org/x/text v0.3.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
)
`,
			src: `package main

import (
	_ "google.golang.org/grpc"
)
`,
			requires: []*modfile.Require{
				{
					Mod: module.Version{
						Path:    "google.golang.org/grpc",
						Version: "v1.40.0",
					},
				},
			},
			expect: `module plugin_module

go 1.17

require google.golang.org/grpc v1.40.0

require (
	github.com/golang/protobuf v1.4.3 // indirect
	golang.org/x/net v0.0.0-20200822124328-c89045814202 // indirect
	golang.org/x/sys v0.0.0-20200323222414-85ca7c5b95cd // indirect
	golang.org/x/text v0.3.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
)
`,
		},
		"do nothing (same version)": {
			gomod: `module plugin_module

go 1.17

require google.golang.org/grpc v1.37.1

require (
	github.com/golang/protobuf v1.4.2 // indirect
	golang.org/x/net v0.0.0-20190311183353-d8887717615a // indirect
	golang.org/x/sys v0.0.0-20190215142949-d0b11bdaac8a // indirect
	golang.org/x/text v0.3.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
)
`,
			src: `package main

import (
	_ "google.golang.org/grpc"
)
`,
			requires: []*modfile.Require{
				{
					Mod: module.Version{
						Path:    "google.golang.org/grpc",
						Version: "v1.37.1",
					},
				},
			},
			expect: `module plugin_module

go 1.17

require google.golang.org/grpc v1.37.1

require (
	github.com/golang/protobuf v1.4.2 // indirect
	golang.org/x/net v0.0.0-20190311183353-d8887717615a // indirect
	golang.org/x/sys v0.0.0-20190215142949-d0b11bdaac8a // indirect
	golang.org/x/text v0.3.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
)
`,
		},
		"add replace": {
			gomod: `module plugin_module

go 1.17

require github.com/zoncoen/scenarigo v0.11.2

require (
	github.com/fatih/color v1.13.0 // indirect
	github.com/goccy/go-yaml v1.9.5 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/zoncoen/query-go v1.1.0 // indirect
	golang.org/x/net v0.0.0-20210813160813-60bc85c4be6d // indirect
	golang.org/x/sys v0.0.0-20211205182925-97ca703d548d // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/xerrors v0.0.0-20220411194840-2f41105eb62f // indirect
	google.golang.org/genproto v0.0.0-20220413183235-5e96e2839df9 // indirect
	google.golang.org/grpc v1.46.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)
`,
			src: `package main

import (
	_ "github.com/zoncoen/scenarigo/protocol/grpc"
)
`,
			requires: []*modfile.Require{
				{
					Mod: module.Version{
						Path:    "google.golang.org/grpc",
						Version: "v1.40.0",
					},
				},
			},
			expect: `module plugin_module

go 1.17

require github.com/zoncoen/scenarigo v0.11.2

require (
	github.com/fatih/color v1.13.0 // indirect
	github.com/goccy/go-yaml v1.9.5 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/zoncoen/query-go v1.1.0 // indirect
	golang.org/x/net v0.0.0-20210813160813-60bc85c4be6d // indirect
	golang.org/x/sys v0.0.0-20211205182925-97ca703d548d // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/xerrors v0.0.0-20220411194840-2f41105eb62f // indirect
	google.golang.org/genproto v0.0.0-20220413183235-5e96e2839df9 // indirect
	google.golang.org/grpc v1.46.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)

replace google.golang.org/grpc v1.46.0 => google.golang.org/grpc v1.40.0
`,
			expectStdout: `WARN: /path/to/go.mod replace google.golang.org/grpc v1.46.0 => v1.40.0
`,
		},
		"overwrite replace": {
			gomod: `module plugin_module

go 1.17

require github.com/zoncoen/scenarigo v0.11.2

require (
	github.com/fatih/color v1.13.0 // indirect
	github.com/goccy/go-yaml v1.9.5 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/zoncoen/query-go v1.1.0 // indirect
	golang.org/x/net v0.0.0-20210813160813-60bc85c4be6d // indirect
	golang.org/x/sys v0.0.0-20211205182925-97ca703d548d // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/xerrors v0.0.0-20220411194840-2f41105eb62f // indirect
	google.golang.org/genproto v0.0.0-20220413183235-5e96e2839df9 // indirect
	google.golang.org/grpc v1.46.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)
`,
			src: `package main

import (
	_ "github.com/zoncoen/scenarigo/protocol/grpc"
)
`,
			requires: []*modfile.Require{
				{
					Mod: module.Version{
						Path:    "google.golang.org/grpc",
						Version: "v1.40.1",
					},
				},
			},
			expect: `module plugin_module

go 1.17

require github.com/zoncoen/scenarigo v0.11.2

require (
	github.com/fatih/color v1.13.0 // indirect
	github.com/goccy/go-yaml v1.9.5 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/zoncoen/query-go v1.1.0 // indirect
	golang.org/x/net v0.0.0-20210813160813-60bc85c4be6d // indirect
	golang.org/x/sys v0.0.0-20211205182925-97ca703d548d // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/xerrors v0.0.0-20220411194840-2f41105eb62f // indirect
	google.golang.org/genproto v0.0.0-20220413183235-5e96e2839df9 // indirect
	google.golang.org/grpc v1.46.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)

replace google.golang.org/grpc v1.46.0 => google.golang.org/grpc v1.40.1
`,
			expectStdout: `WARN: /path/to/go.mod replace google.golang.org/grpc v1.46.0 => v1.40.1
`,
		},
		"do nothing (alredy replaced)": {
			gomod: `module plugin_module

go 1.17

require github.com/zoncoen/scenarigo v0.11.2

require (
	github.com/fatih/color v1.13.0 // indirect
	github.com/goccy/go-yaml v1.9.5 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/zoncoen/query-go v1.1.0 // indirect
	golang.org/x/net v0.0.0-20210813160813-60bc85c4be6d // indirect
	golang.org/x/sys v0.0.0-20211205182925-97ca703d548d // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/xerrors v0.0.0-20220411194840-2f41105eb62f // indirect
	google.golang.org/genproto v0.0.0-20220413183235-5e96e2839df9 // indirect
	google.golang.org/grpc v1.46.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)

replace google.golang.org/grpc v1.46.0 => google.golang.org/grpc v1.40.0
`,
			src: `package main

import (
	_ "github.com/zoncoen/scenarigo/protocol/grpc"
)
`,
			requires: []*modfile.Require{
				{
					Mod: module.Version{
						Path:    "google.golang.org/grpc",
						Version: "v1.40.0",
					},
				},
			},
			expect: `module plugin_module

go 1.17

require github.com/zoncoen/scenarigo v0.11.2

require (
	github.com/fatih/color v1.13.0 // indirect
	github.com/goccy/go-yaml v1.9.5 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/zoncoen/query-go v1.1.0 // indirect
	golang.org/x/net v0.0.0-20210813160813-60bc85c4be6d // indirect
	golang.org/x/sys v0.0.0-20211205182925-97ca703d548d // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/xerrors v0.0.0-20220411194840-2f41105eb62f // indirect
	google.golang.org/genproto v0.0.0-20220413183235-5e96e2839df9 // indirect
	google.golang.org/grpc v1.46.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)

replace google.golang.org/grpc v1.46.0 => google.golang.org/grpc v1.40.0
`,
			expectStdout: "", // don't print the warn log if already replaced
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			gomod := filepath.Join(tmpDir, "go.mod")
			create(t, gomod, test.gomod)
			if test.src != "" {
				create(t, filepath.Join(tmpDir, "main.go"), test.src)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			goCmd, err := findGoCmd(ctx)
			if err != nil {
				t.Fatalf("failed to find go command: %s", err)
			}

			cmd := &cobra.Command{}
			var stdout bytes.Buffer
			cmd.SetOutput(&stdout)
			if err := updateGoMod(cmd, goCmd, gomod, test.requires); err != nil {
				t.Fatalf("failed to update go.mod: %s", err)
			}

			b, err := os.ReadFile(gomod)
			if err != nil {
				t.Fatalf("failed read go.mod: %s", err)
			}
			if got := string(b); got != test.expect {
				dmp := diffmatchpatch.New()
				diffs := dmp.DiffMain(test.expect, got, false)
				t.Errorf("go.mod differs:\n%s", dmp.DiffPrettyText(diffs))
			}

			if got := strings.ReplaceAll(stdout.String(), gomod, "/path/to/go.mod"); got != test.expectStdout {
				dmp := diffmatchpatch.New()
				diffs := dmp.DiffMain(test.expectStdout, got, false)
				t.Errorf("stdout differs:\n%s", dmp.DiffPrettyText(diffs))
			}
		})
	}
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
			filepath.Join(tempDir, e.Name()),
		); err != nil {
			t.Fatalf("failed to rename: %s", err)
		}
	}

	log.Default().SetOutput(io.Discard)
	git := gitkit.NewSSH(gitkit.Config{
		Dir:    tempDir,
		KeyDir: filepath.Join(tempDir, "ssh"),
	})
	git.PublicKeyLookupFunc = func(_ string) (*gitkit.PublicKey, error) {
		return &gitkit.PublicKey{}, nil
	}
	if err := git.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("failed to listen: %s", err)
	}
	go func() {
		_ = git.Serve()
	}()
	t.Cleanup(func() {
		_ = git.Stop()
	})

	u, err := url.Parse(fmt.Sprintf("http://%s", git.Address()))
	if err != nil {
		t.Fatalf("failed to parse URL: %s", err)
	}
	t.Setenv("GIT_SSH_COMMAND", fmt.Sprintf("ssh -p %s -i %s -oStrictHostKeyChecking=no -F /dev/null", u.Port(), filepath.Join(tempDir, "ssh", "gitkit.rsa")))
	t.Setenv("GOPRIVATE", "127.0.0.1")
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
