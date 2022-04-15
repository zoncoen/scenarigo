package plugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"

	"github.com/zoncoen/scenarigo"
	"github.com/zoncoen/scenarigo/cmd/scenarigo/cmd/config"
	"github.com/zoncoen/scenarigo/internal/filepathutil"
	"github.com/zoncoen/scenarigo/version"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "build plugins",
	Long: strings.Trim(`
Builds plugins based on the configuration file.

This command requires go command in $PATH.
`, "\n"),
	Args:          cobra.ExactArgs(0),
	RunE:          build,
	SilenceErrors: true,
	SilenceUsage:  true,
}

var warnColor = color.New(color.Bold, color.FgYellow)

func build(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(config.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if cfg == nil {
		return errors.New("config file not found")
	}

	goCmd, err := findGoCmd(ctx(cmd))
	if err != nil {
		return err
	}

	pluginDir := filepathutil.From(cfg.Root, cfg.PluginDirectory)
	for out, p := range cfg.Plugins {
		src := filepathutil.From(cfg.Root, p.Src)
		if _, err := os.Stat(src); err != nil {
			s, clean, err := downloadModule(ctx(cmd), goCmd, p.Src)
			defer clean()
			if err != nil {
				return fmt.Errorf("failed to build plugin %s: %w", out, err)
			}
			src = s
		}
		if err := buildPlugin(cmd, goCmd, src, filepathutil.From(pluginDir, out), out); err != nil {
			return fmt.Errorf("failed to build plugin %s: %w", out, err)
		}
	}

	return nil
}

func ctx(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}

func findGoCmd(ctx context.Context) (string, error) {
	goVersion := runtime.Version()
	if goCmd, err := exec.LookPath(goVersion); err == nil {
		return goCmd, nil
	}
	goCmd, err := exec.LookPath("go")
	if err != nil {
		return "", fmt.Errorf("go command required: %w", err)
	}
	var stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, goCmd, "version")
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}
	items := strings.Split(stdout.String(), " ")
	if len(items) == 4 {
		if v := items[2]; v != goVersion {
			return "", fmt.Errorf(`required %s but installed %s

You can install the required version of Go by the following commands:

	go install golang.org/dl/%s
	%s download
`, goVersion, v, goVersion, goVersion)
		}
	}
	return goCmd, nil
}

func downloadModule(ctx context.Context, goCmd, mod string) (string, func(), error) {
	tempDir, err := os.MkdirTemp("", "scenarigo-plugin-")
	if err != nil {
		return "", func() {}, fmt.Errorf("failed to create a temporary directory: %w", err)
	}
	goDir := filepath.Join(tempDir, "go")

	envs := []string{
		fmt.Sprintf("GOPATH=%s", goDir),
		fmt.Sprintf("GOMODCACHE=%s", filepath.Join(goDir, "pkg", "mod")),
	}
	clean := func() {
		if err := executeWithEnvs(ctx, envs, tempDir, goCmd, "clean", "-modcache"); err != nil {
			return
		}
		os.RemoveAll(tempDir)
	}

	if err := execute(ctx, tempDir, goCmd, "mod", "init", "main"); err != nil {
		return "", clean, fmt.Errorf("failed to initialize go.mod: %w", err)
	}
	if err := executeWithEnvs(ctx, envs, tempDir, goCmd, downloadCmd(mod)...); err != nil {
		return "", clean, fmt.Errorf("failed to download %s: %w", mod, err)
	}
	if i := strings.Index(mod, "@"); i >= 0 { // trim version
		mod = mod[:i]
	}
	escMod, err := module.EscapePath(mod)
	if err != nil {
		return "", clean, fmt.Errorf("failed to escape path %s: %w", mod, err)
	}
	src := filepath.Join(goDir, "pkg", "mod", escMod)
	src, err = findLatest(filepath.Dir(src), filepath.Base(src))
	if err != nil {
		return "", clean, fmt.Errorf("failed to download %s: %w", mod, err)
	}

	if err := os.Chmod(src, 0o755); err != nil {
		return "", clean, err
	}
	if err := os.Chmod(filepath.Join(src, "go.mod"), 0o644); err != nil {
		if !os.IsNotExist(err) {
			return "", clean, err
		}
	}
	if err := os.Chmod(filepath.Join(src, "go.sum"), 0o644); err != nil {
		if !os.IsNotExist(err) {
			return "", clean, err
		}
	}

	return src, clean, nil
}

func findLatest(dir, base string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}
	for i := len(entries) - 1; i >= 0; i-- {
		if name := entries[i].Name(); strings.HasPrefix(name, base) {
			return filepath.Join(dir, name), nil
		}
	}
	return "", errors.New("not found")
}

func buildPlugin(cmd *cobra.Command, goCmd, src, out, target string) error {
	ctx := ctx(cmd)
	dir := src
	files := []string{}
	if info, err := os.Stat(src); err != nil {
		return fmt.Errorf("failed to find plugin src %s: %w", src, err)
	} else if info.IsDir() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			name := entry.Name()
			if !entry.IsDir() && strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
				files = append(files, name)
			}
		}
	} else {
		dir = filepath.Dir(src)
		files = append(files, src)
	}

	gomodPath := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(gomodPath); err != nil {
		if err := execute(ctx, dir, goCmd, "mod", "init"); err != nil {
			// ref. https://github.com/golang/go/wiki/Modules#why-does-go-mod-init-give-the-error-cannot-determine-module-path-for-source-directory
			if strings.Contains(err.Error(), "cannot determine module path") {
				if err := execute(ctx, dir, goCmd, "mod", "init", "main"); err != nil {
					return fmt.Errorf("failed to initialize go.mod: %w", err)
				}
			} else {
				return fmt.Errorf("failed to initialize go.mod: %w", err)
			}
		}
	}
	if err := updateGoMod(cmd, goCmd, gomodPath); err != nil {
		return err
	}
	if err := execute(ctx, dir, goCmd, "mod", "tidy"); err != nil {
		return fmt.Errorf(`"go mod tidy" failed: %w`, err)
	}

	if err := execute(ctx, dir, goCmd, append([]string{"build", "-buildmode=plugin", "-o", out}, files...)...); err != nil {
		return fmt.Errorf(`"go build -buildmode=plugin -o %s" failed: %w`, out, err)
	}

	return nil
}

func execute(ctx context.Context, wd, name string, args ...string) error {
	return executeWithEnvs(ctx, nil, wd, name, args...)
}

func executeWithEnvs(ctx context.Context, envs []string, wd, name string, args ...string) error {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), envs...)
	if wd != "" {
		cmd.Dir = wd
	}
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return errors.New(strings.TrimSuffix(stderr.String(), "\n"))
	}
	return nil
}

func updateGoMod(cmd *cobra.Command, goCmd, gomodPath string) error {
	goVersion, requires, err := requiredModules()
	if err != nil {
		return err
	}

	if err := editGoMod(cmd, goCmd, gomodPath, func(gomod *modfile.File) error {
		replaces := map[string]string{}
		for _, r := range gomod.Replace {
			replaces[r.Old.Path] = r.Old.Version
		}
		if err := gomod.AddGoStmt(goVersion.Version); err != nil {
			return err
		}
		for _, r := range requires {
			if err := gomod.AddRequire(r.Mod.Path, r.Mod.Version); err != nil {
				return fmt.Errorf("failed to edit %s: %w", gomodPath, err)
			}
			// must use the same module version as scenarigo for building plugins
			if v, ok := replaces[r.Mod.Path]; ok {
				if err := gomod.DropReplace(r.Mod.Path, v); err != nil {
					return fmt.Errorf("failed to edit %s: %w", gomodPath, err)
				}
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to edit require directives: %s", err)
	}

	if err := editGoMod(cmd, goCmd, gomodPath, func(gomod *modfile.File) error {
		current := map[string]string{}
		for _, r := range gomod.Require {
			current[r.Mod.Path] = r.Mod.Version
		}
		for _, r := range requires {
			if v, ok := current[r.Mod.Path]; ok {
				if r.Mod.Version != v {
					fmt.Fprintf(cmd.OutOrStdout(), "%s: %s replace %s %s => %s\n", warnColor.Sprint("WARN"), gomodPath, r.Mod.Path, v, r.Mod.Version)
					if err := gomod.AddReplace(r.Mod.Path, v, r.Mod.Path, r.Mod.Version); err != nil {
						return fmt.Errorf("failed to edit %s: %w", gomodPath, err)
					}
				}
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to edit replace directives: %s", err)
	}

	return nil
}

func editGoMod(cmd *cobra.Command, goCmd, gomodPath string, edit func(*modfile.File) error) error {
	b, err := os.ReadFile(gomodPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", gomodPath, err)
	}
	gomod, err := modfile.Parse(gomodPath, b, nil)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", gomodPath, err)
	}

	if err := edit(gomod); err != nil {
		return fmt.Errorf("failed to edit %s: %w", gomodPath, err)
	}
	gomod.Cleanup()
	edited, err := gomod.Format()
	if err != nil {
		return fmt.Errorf("failed to edit %s: %w", gomodPath, err)
	}

	f, err := os.Create(gomodPath)
	if err != nil {
		return fmt.Errorf("failed to edit %s: %w", gomodPath, err)
	}
	defer f.Close()
	if _, err := f.Write(edited); err != nil {
		return fmt.Errorf("failed to edit %s: %w", gomodPath, err)
	}
	if err := execute(ctx(cmd), filepath.Dir(gomodPath), goCmd, "mod", "tidy"); err != nil {
		return fmt.Errorf(`"go mod tidy" failed: %w`, err)
	}

	return nil
}

func requiredModules() (*modfile.Go, []*modfile.Require, error) {
	gomod, err := modfile.Parse("go.mod", scenarigo.GoModBytes, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse go.mod of scenarigo: %s", err)
	}
	if v := version.String(); !strings.HasSuffix(v, "-dev") {
		return gomod.Go, append([]*modfile.Require{{
			Mod: module.Version{
				Path:    "github.com/zoncoen/scenarigo",
				Version: v,
			},
		}}, gomod.Require...), nil
	}
	return gomod.Go, gomod.Require, nil
}
