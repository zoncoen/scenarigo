package plugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/build"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"

	"github.com/zoncoen/scenarigo"
	"github.com/zoncoen/scenarigo/cmd/scenarigo/cmd/config"
	"github.com/zoncoen/scenarigo/internal/filepathutil"
	"github.com/zoncoen/scenarigo/version"
)

var gomodVer string

func init() {
	gomod, err := modfile.Parse("go.mod", scenarigo.GoModBytes, nil)
	if err != nil {
		panic(fmt.Errorf("failed to parse go.mod of scenarigo: %w", err))
	}
	gomodVer = gomod.Go.Version
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "build plugins",
	Long: strings.Trim(`
Builds plugins based on the configuration file.

This command requires go command in $PATH.
`, "\n"),
	Args:          cobra.ExactArgs(0),
	RunE:          buildRun,
	SilenceErrors: true,
	SilenceUsage:  true,
}

var warnColor = color.New(color.Bold, color.FgYellow)

func buildRun(cmd *cobra.Command, args []string) error {
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

	pbs := make([]*pluginBuilder, 0, len(cfg.Plugins))
	pluginModules := map[string]*modfile.Require{}
	pluginDir := filepathutil.From(cfg.Root, cfg.PluginDirectory)
	for out, p := range cfg.Plugins {
		mod := filepathutil.From(cfg.Root, p.Src)
		var src string
		if _, err := os.Stat(mod); err != nil {
			m, s, r, clean, err := downloadModule(ctx(cmd), goCmd, p.Src)
			defer clean()
			if err != nil {
				return fmt.Errorf("failed to build plugin %s: %w", out, err)
			}
			mod = m
			src = s
			pluginModules[r.Mod.Path] = r
		}
		// NOTE: All module names must be unique and different from the standard modules.
		defaultModName := filepath.Join("plugins", strings.TrimSuffix(out, ".so"))
		pb, err := newPluginBuilder(cmd, goCmd, mod, src, filepathutil.From(pluginDir, out), defaultModName)
		if err != nil {
			return fmt.Errorf("failed to build plugin %s: %w", out, err)
		}
		pbs = append(pbs, pb)
	}

	// maximum version selection
	overrides := map[string]*modfile.Require{}
	for _, pb := range pbs {
		for _, r := range pb.gomod.Require {
			o, ok := overrides[r.Mod.Path]
			if !ok {
				overrides[r.Mod.Path] = r
				continue
			}
			if semver.Compare(o.Mod.Version, r.Mod.Version) < 0 {
				overrides[r.Mod.Path] = r
			}
		}
	}

	for m, r := range pluginModules {
		overrides[m] = r
	}

	requires, err := requiredModulesByScenarigo()
	if err != nil {
		return err
	}
	for _, r := range requires {
		overrides[r.Mod.Path] = r
	}

	for _, pb := range pbs {
		if err := pb.build(cmd, goCmd, overrides); err != nil {
			return fmt.Errorf("failed to build plugin %s: %w", pb.out, err)
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
	goCmd, err := exec.LookPath("go")
	var verr error
	if err == nil {
		verr = checkGoVersion(ctx, goCmd, goVersion)
		if verr == nil {
			return goCmd, nil
		}
	}
	if goCmd, err := exec.LookPath(goVersion); err == nil {
		if err := checkGoVersion(ctx, goCmd, goVersion); err == nil {
			return goCmd, nil
		}
	}
	if err == nil {
		return "", verr
	}
	return "", fmt.Errorf("go command required: %w", err)
}

func checkGoVersion(ctx context.Context, goCmd, ver string) error {
	var stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, goCmd, "version")
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return err
	}
	items := strings.Split(stdout.String(), " ")
	if len(items) != 4 {
		return errors.New("invalid version output or scenarigo bug")
	}
	if v := items[2]; v != ver {
		// nolint:revive
		return fmt.Errorf(`required %s but installed %s

You can install the required version of Go by the following commands:

	go install golang.org/dl/%s@latest
	%s download
`, ver, v, ver, ver)
	}
	return nil
}

func downloadModule(ctx context.Context, goCmd, p string) (string, string, *modfile.Require, func(), error) {
	tempDir, err := os.MkdirTemp("", "scenarigo-plugin-")
	if err != nil {
		return "", "", nil, func() {}, fmt.Errorf("failed to create a temporary directory: %w", err)
	}

	clean := func() {
		os.RemoveAll(tempDir)
	}

	if err := execute(ctx, tempDir, goCmd, "mod", "init", "download_module"); err != nil {
		return "", "", nil, clean, fmt.Errorf("failed to initialize go.mod: %w", err)
	}
	if err := execute(ctx, tempDir, goCmd, downloadCmd(p)...); err != nil {
		return "", "", nil, clean, fmt.Errorf("failed to download %s: %w", p, err)
	}
	mod, src, req, err := modSrcPath(tempDir, p)
	if err != nil {
		return "", "", nil, clean, fmt.Errorf("failed to get module path: %w", err)
	}

	clean = func() {
		os.RemoveAll(tempDir)
		// The downloaded plugin module should be removed because its go.mod may have been modified.
		if err := filepath.Walk(mod, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			return os.Chmod(path, 0o777)
		}); err == nil {
			os.RemoveAll(mod)
		}
	}

	if err := os.Chmod(mod, 0o755); err != nil {
		return "", "", nil, clean, err
	}
	if err := os.Chmod(filepath.Join(mod, "go.mod"), 0o644); err != nil {
		if !os.IsNotExist(err) {
			return "", "", nil, clean, err
		}
	}
	if err := os.Chmod(filepath.Join(mod, "go.sum"), 0o644); err != nil {
		if !os.IsNotExist(err) {
			return "", "", nil, clean, err
		}
	}

	return mod, src, req, clean, nil
}

func modSrcPath(tempDir, mod string) (string, string, *modfile.Require, error) {
	if i := strings.Index(mod, "@"); i >= 0 { // trim version
		mod = mod[:i]
	}
	b, err := os.ReadFile(filepath.Join(tempDir, "go.mod"))
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to read file: %w", err)
	}
	gomod, err := modfile.Parse("go.mod", b, nil)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to parse go.mod: %w", err)
	}
	parts := strings.Split(mod, "/")
	for i := len(parts); i > 1; i-- {
		m := strings.Join(parts[:i], "/")
		for _, r := range gomod.Require {
			if r.Mod.Path == m {
				p, err := module.EscapePath(r.Mod.Path)
				if err != nil {
					return "", "", nil, fmt.Errorf("failed to escape module path %s: %w", r.Mod.Path, err)
				}
				return filepath.Join(
					build.Default.GOPATH, "pkg", "mod",
					fmt.Sprintf("%s@%s", p, r.Mod.Version),
				), filepath.Join(parts[i:]...), r, nil
			}
		}
	}
	return "", "", nil, errors.New("module not found on go.mod")
}

type pluginBuilder struct {
	dir       string
	src       string
	gomodPath string
	gomod     *modfile.File
	out       string
}

func newPluginBuilder(cmd *cobra.Command, goCmd, mod, src, out, defaultModName string) (*pluginBuilder, error) {
	ctx := ctx(cmd)
	dir := mod
	info, err := os.Stat(mod)
	if err != nil {
		return nil, fmt.Errorf("failed to find plugin src %s: %w", mod, err)
	}
	if !info.IsDir() {
		dir, src = filepath.Split(mod)
	}
	src = fmt.Sprintf(".%c%s", filepath.Separator, src) // modify the path to explicit relative

	gomodPath := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(gomodPath); err != nil {
		if err := execute(ctx, dir, goCmd, "mod", "init"); err != nil {
			// ref. https://github.com/golang/go/wiki/Modules#why-does-go-mod-init-give-the-error-cannot-determine-module-path-for-source-directory
			if strings.Contains(err.Error(), "cannot determine module path") {
				if err := execute(ctx, dir, goCmd, "mod", "init", defaultModName); err != nil {
					return nil, fmt.Errorf("failed to initialize go.mod: %w", err)
				}
			} else {
				return nil, fmt.Errorf("failed to initialize go.mod: %w", err)
			}
		}
	}

	if err := execute(ctx, dir, goCmd, "mod", "tidy"); err != nil {
		return nil, fmt.Errorf(`"go mod tidy" failed: %w`, err)
	}

	b, err := os.ReadFile(gomodPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", gomodPath, err)
	}
	gomod, err := modfile.Parse(gomodPath, b, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", gomodPath, err)
	}

	return &pluginBuilder{
		dir:       dir,
		src:       src,
		gomodPath: gomodPath,
		gomod:     gomod,
		out:       out,
	}, nil
}

func (pb *pluginBuilder) build(cmd *cobra.Command, goCmd string, overrides map[string]*modfile.Require) error {
	ctx := ctx(cmd)
	if err := updateGoMod(cmd, goCmd, pb.gomodPath, overrides); err != nil {
		return err
	}
	if err := execute(ctx, pb.dir, goCmd, "build", "-buildmode=plugin", "-o", pb.out, pb.src); err != nil {
		return fmt.Errorf(`"go build -buildmode=plugin -o %s %s" failed: %w`, pb.out, pb.src, err)
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

func updateGoMod(cmd *cobra.Command, goCmd, gomodPath string, overrides map[string]*modfile.Require) error {
	requireKeys := []string{}
	requires := map[string]modfile.Require{}
	replaces := map[string]modfile.Replace{}
	overrideKeys := []string{}
	if err := editGoMod(cmd, goCmd, gomodPath, func(gomod *modfile.File) error {
		for _, r := range gomod.Require {
			requireKeys = append(requireKeys, r.Mod.Path)
			requires[r.Mod.Path] = *r
		}
		for _, r := range gomod.Replace {
			replaces[r.Old.Path] = *r
		}
		if err := gomod.AddGoStmt(gomodVer); err != nil {
			return fmt.Errorf("failed to edit %s: %w", gomodPath, err)
		}
		for _, r := range overrides {
			overrideKeys = append(overrideKeys, r.Mod.Path)
			if err := gomod.AddRequire(r.Mod.Path, r.Mod.Version); err != nil {
				return fmt.Errorf("failed to edit %s: %w", gomodPath, err)
			}
			// must use the same module version as scenarigo for building plugins
			if v, ok := replaces[r.Mod.Path]; ok {
				if err := gomod.DropReplace(r.Mod.Path, v.Old.Version); err != nil {
					return fmt.Errorf("failed to edit %s: %w", gomodPath, err)
				}
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to edit require directives: %w", err)
	}
	sort.Strings(requireKeys)
	sort.Strings(overrideKeys)

	if err := editGoMod(cmd, goCmd, gomodPath, func(gomod *modfile.File) error {
		current := map[string]string{}
		for _, r := range gomod.Require {
			current[r.Mod.Path] = r.Mod.Version
		}
		for _, k := range requireKeys {
			r := requires[k]
			if v, ok := current[r.Mod.Path]; ok {
				if r.Mod.Version != v {
					fmt.Fprintf(cmd.OutOrStdout(), "%s: %s require %s %s => %s\n", warnColor.Sprint("WARN"), gomodPath, r.Mod.Path, r.Mod.Version, v)
				}
			}
		}
		for _, k := range overrideKeys {
			r := overrides[k]
			if v, ok := current[r.Mod.Path]; ok {
				if r.Mod.Version != v {
					if replaced, ok := replaces[r.Mod.Path]; !ok || replaced.New.Path != r.Mod.Path || replaced.New.Version != r.Mod.Version {
						fmt.Fprintf(cmd.OutOrStdout(), "%s: %s replace %s %s => %s\n", warnColor.Sprint("WARN"), gomodPath, r.Mod.Path, v, r.Mod.Version)
					}
					if err := gomod.AddReplace(r.Mod.Path, v, r.Mod.Path, r.Mod.Version); err != nil {
						return fmt.Errorf("failed to edit %s: %w", gomodPath, err)
					}
				}
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to edit replace directives: %w", err)
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

func requiredModulesByScenarigo() ([]*modfile.Require, error) {
	gomod, err := modfile.Parse("go.mod", scenarigo.GoModBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod of scenarigo: %w", err)
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
