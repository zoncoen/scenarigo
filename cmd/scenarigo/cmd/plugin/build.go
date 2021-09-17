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

	"github.com/spf13/cobra"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"

	"github.com/zoncoen/scenarigo"
	"github.com/zoncoen/scenarigo/cmd/scenarigo/cmd/config"
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

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get the current directory: %w", err)
	}
	defer func() {
		os.Chdir(wd)
	}()

	pluginDir := pathFromRoot(cfg.Root, cfg.PluginDirectory)
	for out, p := range cfg.Plugins {
		if err := buildPlugin(ctx(cmd), goCmd, pathFromRoot(cfg.Root, p.Src), pathFromRoot(pluginDir, out), out); err != nil {
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

	go get golang.org/dl/%s
	%s download
`, goVersion, v, goVersion, goVersion)
		}
	}
	return goCmd, nil
}

func buildPlugin(ctx context.Context, goCmd, src, out, target string) error {
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
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("failed to change working directory: %w", err)
	}

	gomodPath := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(gomodPath); err != nil {
		if err := execute(ctx, goCmd, "mod", "init"); err != nil {
			// ref. https://github.com/golang/go/wiki/Modules#why-does-go-mod-init-give-the-error-cannot-determine-module-path-for-source-directory
			if strings.Contains(err.Error(), "cannot determine module path") {
				if err := execute(ctx, goCmd, "mod", "init", "main"); err != nil {
					return fmt.Errorf("failed to initialize go.mod: %w", err)
				}
			} else {
				return fmt.Errorf("failed to initialize go.mod: %w", err)
			}
		}
	}
	if err := addRequires(gomodPath); err != nil {
		return err
	}
	if err := execute(ctx, goCmd, "mod", "tidy"); err != nil {
		return fmt.Errorf(`"go mod tidy" failed: %w`, err)
	}

	if err := execute(ctx, goCmd, append([]string{"build", "-buildmode=plugin", "-o", out}, files...)...); err != nil {
		return fmt.Errorf(`"go build -buildmode=plugin -o %s" failed: %w`, out, err)
	}

	return nil
}

func pathFromRoot(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

func execute(ctx context.Context, name string, args ...string) error {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return errors.New(strings.TrimSuffix(stderr.String(), "\n"))
	}
	return nil
}

func addRequires(gomodPath string) error {
	b, err := os.ReadFile(gomodPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", gomodPath, err)
	}
	gomod, err := modfile.Parse(gomodPath, b, nil)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", gomodPath, err)
	}
	requires, err := requiredModules()
	if err != nil {
		return err
	}
	for _, r := range requires {
		if err := gomod.AddRequire(r.Mod.Path, r.Mod.Version); err != nil {
			return fmt.Errorf("failed to edit %s: %w", gomodPath, err)
		}
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
	return nil
}

func requiredModules() ([]*modfile.Require, error) {
	gomod, err := modfile.Parse("go.mod", scenarigo.GoModBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod of scenarigo: %s", err)
	}
	if v := version.String(); !strings.HasSuffix(v, "-dev") {
		return append([]*modfile.Require{{
			Mod: module.Version{
				Path:    "github.com/zoncoen/scenarigo",
				Version: v,
			},
		}}, gomod.Require...), nil
	}
	return gomod.Require, nil
}
