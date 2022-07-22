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

type overrideModule struct {
	require    *modfile.Require
	requiredBy string
	count      int
	replace    *modfile.Replace
	replacedBy string
}

func (o *overrideModule) requireReplace() (*modfile.Require, string, *modfile.Replace, string) {
	if o.replace != nil {
		if o.require == nil || o.replace.Old.Path == o.replace.New.Path {
			return &modfile.Require{
				Mod:      o.replace.New,
				Indirect: false,
				Syntax:   nil,
			}, o.replacedBy, nil, ""
		}
	}
	return o.require, o.requiredBy, o.replace, o.replacedBy
}

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
	pluginModules := map[string]*overrideModule{}
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
			pluginModules[r.Mod.Path] = &overrideModule{
				require:    r,
				requiredBy: out,
				count:      1,
				replace:    nil,
				replacedBy: "",
			}
		}
		// NOTE: All module names must be unique and different from the standard modules.
		defaultModName := filepath.Join("plugins", strings.TrimSuffix(out, ".so"))
		pb, err := newPluginBuilder(cmd, goCmd, out, mod, src, filepathutil.From(pluginDir, out), defaultModName)
		if err != nil {
			return fmt.Errorf("failed to build plugin %s: %w", out, err)
		}
		pbs = append(pbs, pb)
	}

	overrides, err := selectUnifiedVersions(pbs)
	if err != nil {
		return fmt.Errorf("failed to build plugin: %w", err)
	}

	for m, o := range pluginModules {
		overrides[m] = &overrideModule{
			require:    o.require,
			requiredBy: o.requiredBy,
			count:      1,
			replace:    nil,
			replacedBy: "",
		}
	}

	requires, err := requiredModulesByScenarigo()
	if err != nil {
		return err
	}
	for _, r := range requires {
		overrides[r.Mod.Path] = &overrideModule{
			require:    r,
			requiredBy: "scenarigo",
			count:      1,
			replace:    nil,
			replacedBy: "",
		}
	}

	for _, pb := range pbs {
		if err := pb.build(cmd, goCmd, overrides); err != nil {
			return fmt.Errorf("failed to build plugin %s: %w", pb.name, err)
		}
	}

	return nil
}

func selectUnifiedVersions(pbs []*pluginBuilder) (map[string]*overrideModule, error) {
	overrides := map[string]*overrideModule{}
	for _, pb := range pbs {
		// maximum version selection
		for _, r := range pb.gomod.Require {
			o, ok := overrides[r.Mod.Path]
			if !ok {
				overrides[r.Mod.Path] = &overrideModule{
					require:    r,
					requiredBy: pb.name,
					count:      1,
					replace:    nil,
					replacedBy: "",
				}
				continue
			}
			overrides[r.Mod.Path].count++
			if semver.Compare(o.require.Mod.Version, r.Mod.Version) < 0 {
				overrides[r.Mod.Path].require = r
				overrides[r.Mod.Path].requiredBy = pb.name
			}
		}
		for _, r := range pb.gomod.Replace {
			o, ok := overrides[r.Old.Path]
			if !ok {
				overrides[r.Old.Path] = &overrideModule{
					replace:    r,
					replacedBy: pb.name,
				}
				continue
			}
			if o.replace != nil {
				if o.replace.New.Path != r.New.Path || o.replace.New.Version != r.New.Version {
					return nil, fmt.Errorf("%s: replace %s directive conflicts: %s => %s %s, %s => %s %s", pb.name, r.Old.Path, o.replacedBy, o.replace.New.Path, o.replace.New.Version, pb.name, r.New.Path, r.New.Version)
				}
			}
			o.replace = r
			o.replacedBy = pb.name
			overrides[r.Old.Path] = o
		}
	}
	return overrides, nil
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
	name      string
	dir       string
	src       string
	gomodPath string
	gomod     *modfile.File
	out       string
}

func newPluginBuilder(cmd *cobra.Command, goCmd, name, mod, src, out, defaultModName string) (*pluginBuilder, error) {
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
		name:      name,
		dir:       dir,
		src:       src,
		gomodPath: gomodPath,
		gomod:     gomod,
		out:       out,
	}, nil
}

func (pb *pluginBuilder) build(cmd *cobra.Command, goCmd string, overrides map[string]*overrideModule) error {
	ctx := ctx(cmd)
	if err := updateGoMod(cmd, goCmd, pb.name, pb.gomodPath, overrides); err != nil {
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

func updateGoMod(cmd *cobra.Command, goCmd, name, gomodPath string, overrides map[string]*overrideModule) error {
	initialRequires, initialReplaces, overrideKeys, err := updateRequireDirectives(cmd, goCmd, gomodPath, overrides)
	if err != nil {
		return err
	}
	if err := updateReplaceDirectives(cmd, goCmd, gomodPath, overrides, overrideKeys); err != nil {
		return err
	}
	if err := printUpdatedResult(cmd, goCmd, name, gomodPath, overrides, initialRequires, initialReplaces); err != nil {
		return err
	}
	return nil
}

func updateRequireDirectives(cmd *cobra.Command, goCmd, gomodPath string, overrides map[string]*overrideModule) (map[string]modfile.Require, map[string]modfile.Replace, []string, error) {
	initialRequires := map[string]modfile.Require{}
	initialReplaces := map[string]modfile.Replace{}
	overrideKeys := []string{}
	if err := editGoMod(cmd, goCmd, gomodPath, func(gomod *modfile.File) error {
		for _, r := range gomod.Require {
			initialRequires[r.Mod.Path] = *r
		}
		for _, r := range gomod.Replace {
			initialReplaces[r.Old.Path] = *r
		}
		if semver.Compare(gomod.Go.Version, gomodVer) < 0 {
			if err := gomod.AddGoStmt(gomodVer); err != nil {
				return fmt.Errorf("%s: %w", gomodPath, err)
			}
		}
		for _, o := range overrides {
			require, _, _, _ := o.requireReplace()
			overrideKeys = append(overrideKeys, require.Mod.Path)
			if err := gomod.AddRequire(require.Mod.Path, require.Mod.Version); err != nil {
				return fmt.Errorf("%s: %w", gomodPath, err)
			}
		}
		return nil
	}); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to edit require directive: %w", err)
	}
	sort.Strings(overrideKeys)
	return initialRequires, initialReplaces, overrideKeys, nil
}

func updateReplaceDirectives(cmd *cobra.Command, goCmd, gomodPath string, overrides map[string]*overrideModule, overrideKeys []string) error {
	if err := editGoMod(cmd, goCmd, gomodPath, func(gomod *modfile.File) error {
		requires := map[string]string{}
		for _, r := range gomod.Require {
			requires[r.Mod.Path] = r.Mod.Version
		}
		replaces := map[string]modfile.Replace{}
		for _, r := range gomod.Replace {
			if _, ok := requires[r.Old.Path]; !ok {
				if err := gomod.DropReplace(r.Old.Path, r.Old.Version); err != nil {
					return fmt.Errorf("%s: %w", gomodPath, err)
				}
				continue
			}
			replaces[r.Old.Path] = *r
		}
		for _, k := range overrideKeys {
			o := overrides[k]
			require, _, replace, _ := o.requireReplace()
			if v, ok := replaces[require.Mod.Path]; ok {
				if err := gomod.DropReplace(require.Mod.Path, v.Old.Version); err != nil {
					return fmt.Errorf("%s: %w", gomodPath, err)
				}
			}
			if replace != nil {
				if v, ok := requires[replace.Old.Path]; ok {
					if err := gomod.AddReplace(replace.Old.Path, v, replace.New.Path, replace.New.Version); err != nil {
						return fmt.Errorf("%s: %w", gomodPath, err)
					}
				}
			} else {
				if v, ok := requires[require.Mod.Path]; ok {
					if require.Mod.Version != v {
						if err := gomod.AddReplace(require.Mod.Path, v, require.Mod.Path, require.Mod.Version); err != nil {
							return fmt.Errorf("%s: %w", gomodPath, err)
						}
					}
				}
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to edit replace directive: %w", err)
	}
	return nil
}

type requireDiff struct {
	old modfile.Require
	new modfile.Require
}

type replaceDiff struct {
	old modfile.Replace
	new modfile.Replace
}

func printUpdatedResult(cmd *cobra.Command, goCmd, name, gomodPath string, overrides map[string]*overrideModule, initialRequires map[string]modfile.Require, initialReplaces map[string]modfile.Replace) error {
	gomod, err := parseGoMod(cmd, goCmd, gomodPath)
	if err != nil {
		return err
	}
	printUpdatedRequires(cmd, name, overrides, initialRequires, gomod)
	printUpdatedReplaces(cmd, name, overrides, initialReplaces, gomod)
	return nil
}

func printUpdatedRequires(cmd *cobra.Command, name string, overrides map[string]*overrideModule, initialRequires map[string]modfile.Require, gomod *modfile.File) {
	requireKeys := []string{}
	requireDiffs := map[string]*requireDiff{}
	for _, r := range initialRequires {
		requireKeys = append(requireKeys, r.Mod.Path)
		requireDiffs[r.Mod.Path] = &requireDiff{
			old: r,
		}
	}
	for _, r := range gomod.Require {
		diff, ok := requireDiffs[r.Mod.Path]
		if ok {
			diff.new = *r
		} else {
			requireKeys = append(requireKeys, r.Mod.Path)
			requireDiffs[r.Mod.Path] = &requireDiff{
				new: *r,
			}
		}
	}
	sort.Strings(requireKeys)

	for _, k := range requireKeys {
		diff := requireDiffs[k]
		switch {
		case diff.old.Mod.Path == "":
			if !diff.new.Indirect {
				if o := overrides[k]; o != nil {
					_, requiredBy, _, _ := o.requireReplace()
					fmt.Fprintf(cmd.OutOrStdout(), "%s: %s: add require %s %s by %s\n", warnColor.Sprint("WARN"), name, k, diff.new.Mod.Version, requiredBy)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "%s: %s: add require %s %s\n", warnColor.Sprint("WARN"), name, k, diff.new.Mod.Version)
				}
			}
		case diff.new.Mod.Path == "":
			if !diff.old.Indirect {
				fmt.Fprintf(cmd.OutOrStdout(), "%s: %s: remove require %s %s\n", warnColor.Sprint("WARN"), name, k, diff.old.Mod.Version)
			}
		case diff.old.Mod.Version != diff.new.Mod.Version:
			if !diff.old.Indirect || !diff.new.Indirect {
				if o := overrides[k]; o != nil {
					_, requiredBy, _, _ := o.requireReplace()
					fmt.Fprintf(cmd.OutOrStdout(), "%s: %s: change require %s %s ==> %s by %s\n", warnColor.Sprint("WARN"), name, k, diff.old.Mod.Version, diff.new.Mod.Version, requiredBy)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "%s: %s: change require %s %s ==> %s\n", warnColor.Sprint("WARN"), name, k, diff.old.Mod.Version, diff.new.Mod.Version)
				}
			}
		}
	}
}

func printUpdatedReplaces(cmd *cobra.Command, name string, overrides map[string]*overrideModule, initialReplaces map[string]modfile.Replace, gomod *modfile.File) {
	replaceKeys := []string{}
	replaceDiffs := map[string]*replaceDiff{}
	for _, r := range initialReplaces {
		replaceKeys = append(replaceKeys, r.Old.Path)
		replaceDiffs[r.Old.Path] = &replaceDiff{
			old: r,
		}
	}
	for _, r := range gomod.Replace {
		diff, ok := replaceDiffs[r.Old.Path]
		if ok {
			diff.new = *r
		} else {
			replaceKeys = append(replaceKeys, r.Old.Path)
			replaceDiffs[r.Old.Path] = &replaceDiff{
				new: *r,
			}
		}
	}
	sort.Strings(replaceKeys)

	for _, k := range replaceKeys {
		diff := replaceDiffs[k]
		switch {
		case diff.old.Old.Path == "":
			if o := overrides[k]; o != nil {
				_, by, replace, replaceBy := o.requireReplace()
				if replace != nil {
					by = replaceBy
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s: %s: add replace %s %s => %s %s by %s\n", warnColor.Sprint("WARN"), name, k, diff.new.Old.Version, diff.new.New.Path, diff.new.New.Version, by)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "%s: %s: add replace %s %s => %s %s\n", warnColor.Sprint("WARN"), name, k, diff.new.Old.Version, diff.new.New.Path, diff.new.New.Version)
			}
		case diff.new.Old.Path == "":
			fmt.Fprintf(cmd.OutOrStdout(), "%s: %s: remove replace %s %s => %s %s\n", warnColor.Sprint("WARN"), name, k, diff.old.Old.Version, diff.old.New.Path, diff.old.New.Version)
		case diff.old.New.Path != diff.new.New.Path || diff.old.New.Version != diff.new.New.Version:
			if o := overrides[k]; o != nil {
				_, by, replace, replaceBy := o.requireReplace()
				if replace != nil {
					by = replaceBy
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s: %s: change replace %s %s => %s %s ==> %s %s => %s %s by %s\n", warnColor.Sprint("WARN"), name, k, diff.old.Old.Version, diff.old.New.Path, diff.old.New.Version, k, diff.new.Old.Version, diff.new.New.Path, diff.new.New.Version, by)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "%s: %s: change replace %s %s => %s %s ==> %s %s => %s %s\n", warnColor.Sprint("WARN"), name, k, diff.old.Old.Version, diff.old.New.Path, diff.old.New.Version, k, diff.new.Old.Version, diff.new.New.Path, diff.new.New.Version)
			}
		}
	}
}

func editGoMod(cmd *cobra.Command, goCmd, gomodPath string, edit func(*modfile.File) error) error {
	gomod, err := parseGoMod(cmd, goCmd, gomodPath)
	if err != nil {
		return err
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

func parseGoMod(cmd *cobra.Command, goCmd, gomodPath string) (*modfile.File, error) {
	b, err := os.ReadFile(gomodPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", gomodPath, err)
	}
	gomod, err := modfile.Parse(gomodPath, b, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", gomodPath, err)
	}
	return gomod, nil
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
