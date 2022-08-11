package scenarigo

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/fatih/color"
	"github.com/pkg/errors"

	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/plugin"
	"github.com/zoncoen/scenarigo/protocol/grpc"
	"github.com/zoncoen/scenarigo/protocol/http"
	"github.com/zoncoen/scenarigo/reporter"
	"github.com/zoncoen/scenarigo/schema"
)

func init() {
	http.Register()
	grpc.Register()
}

// Runner represents a test runner.
type Runner struct {
	pluginDir       *string
	plugins         map[string]schema.PluginConfig
	scenarioFiles   []string
	scenarioReaders []io.Reader
	enabledColor    bool
	rootDir         string
	reportConfig    schema.ReportConfig
}

// NewRunner returns a new test runner.
func NewRunner(opts ...func(*Runner) error) (*Runner, error) {
	r := &Runner{} //nolint:exhaustruct
	r.enabledColor = !color.NoColor
	for _, opt := range opts {
		if err := opt(r); err != nil {
			return nil, err
		}
	}
	if r.rootDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		r.rootDir = wd
	}
	return r, nil
}

// WithConfig returns a option which sets configuration.
func WithConfig(config *schema.Config) func(*Runner) error {
	return func(r *Runner) error {
		if config == nil {
			return nil
		}

		r.rootDir = config.Root
		scenarios := make([]string, len(config.Scenarios))
		for i, s := range config.Scenarios {
			scenarios[i] = filepath.Join(r.rootDir, s)
		}

		var opts []func(r *Runner) error
		opts = append(opts, WithScenarios(scenarios...))
		if config.PluginDirectory != "" {
			opts = append(opts, WithPluginDir(filepath.Join(r.rootDir, config.PluginDirectory)))
		}
		for _, opt := range opts {
			if err := opt(r); err != nil {
				return err
			}
		}
		r.plugins = config.Plugins
		if config.Output.Colored != nil {
			r.enabledColor = *config.Output.Colored
		}
		r.reportConfig = config.Output.Report
		return nil
	}
}

// WithScenarios returns a option which finds and sets test scenario files.
func WithScenarios(paths ...string) func(*Runner) error {
	return func(r *Runner) error {
		for i, path := range paths {
			abs, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("failed to find test scenarios: %w", err)
			}
			paths[i] = abs
		}
		files, err := getAllFiles(paths...)
		if err != nil {
			return fmt.Errorf("failed to find test scenarios: %w", err)
		}
		r.scenarioFiles = files
		return nil
	}
}

// WithPluginDir returns a option which sets plugin root directory.
func WithPluginDir(path string) func(*Runner) error {
	return func(r *Runner) error {
		abs, err := filepath.Abs(path)
		if err != nil {
			return errors.Wrapf(err, `failed to set plugin directory "%s"`, path)
		}
		r.pluginDir = &abs
		return nil
	}
}

// WithScenariosFromReader returns a option which sets readers to read scenario contents.
func WithScenariosFromReader(readers ...io.Reader) func(*Runner) error {
	return func(r *Runner) error {
		r.scenarioReaders = readers
		return nil
	}
}

// WithOptionsFromEnv returns a option which sets flag whether accepts configuration from ENV.
// Currently Available ENV variables are the following.
//     - SCENARIGO_COLOR=(1|true|TRUE)
func WithOptionsFromEnv(isEnv bool) func(*Runner) error {
	return func(r *Runner) error {
		if isEnv {
			r.setOptionsFromEnv()
		}
		return nil
	}
}

var yamlPattern = regexp.MustCompile(`(?i)\.ya?ml$`)

func looksLikeYAML(path string) bool {
	return yamlPattern.MatchString(path)
}

func getAllFiles(paths ...string) ([]string, error) {
	files := []string{}
	for _, path := range paths {
		if err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if looksLikeYAML(path) {
				files = append(files, path)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return files, nil
}

const (
	envScenarigoColor = "SCENARIGO_COLOR"
)

func (r *Runner) setOptionsFromEnv() {
	r.setEnabledColor(os.Getenv(envScenarigoColor))
}

func (r *Runner) setEnabledColor(envColor string) {
	if envColor == "" {
		return
	}
	result, _ := strconv.ParseBool(envColor)
	r.enabledColor = result
}

// ScenarioFiles returns all scenario file paths.
func (r *Runner) ScenarioFiles() []string {
	return r.scenarioFiles
}

// Run runs all tests.
func (r *Runner) Run(ctx *context.Context) {
	// setup context
	if r.pluginDir != nil {
		ctx = ctx.WithPluginDir(*r.pluginDir)
	}
	ctx = ctx.WithEnabledColor(r.enabledColor)

	// open plugins
	pluginDir := r.rootDir
	if dir := ctx.PluginDir(); dir != "" {
		pluginDir = dir
	}
	sm := make(setupMap)
	for out := range r.plugins {
		p, err := plugin.Open(filepath.Join(pluginDir, out))
		if err != nil {
			sm[out] = func(ctx *context.Context) (*context.Context, func(*context.Context)) {
				ctx.Reporter().Fatalf("failed to open plugin: %s", err)
				return nil, nil
			}
			continue
		}
		if setup := p.GetSetup(); setup != nil {
			sm[out] = setup
		}
	}
	ctx, teardown := sm.setup(ctx)
	if ctx.Reporter().Failed() {
		teardown(ctx)
		return
	}

	for _, f := range r.scenarioFiles {
		testName, err := filepath.Rel(r.rootDir, f)
		if err != nil {
			testName = f
		}
		ctx.Run(testName, func(ctx *context.Context) {
			scns, err := schema.LoadScenarios(f)
			if err != nil {
				ctx.Reporter().Fatalf("failed to load scenarios: %s", err)
			}
			for _, scn := range scns {
				scn := scn
				ctx = ctx.WithNode(scn.Node)
				ctx.Run(scn.Title, func(ctx *context.Context) {
					ctx.Reporter().Parallel()
					_ = RunScenario(ctx, scn)
				})
			}
		})
	}
	for i, reader := range r.scenarioReaders {
		ctx.Run(fmt.Sprint(i), func(ctx *context.Context) {
			scns, err := schema.LoadScenariosFromReader(reader)
			if err != nil {
				ctx.Reporter().Fatalf("failed to load scenarios: %s", err)
			}
			for _, scn := range scns {
				scn := scn
				ctx = ctx.WithNode(scn.Node)
				ctx.Run(scn.Title, func(ctx *context.Context) {
					ctx.Reporter().Parallel()
					_ = RunScenario(ctx, scn)
				})
			}
		})
	}
	teardown(ctx)
	r.writeTestReport(ctx)
}

func (r *Runner) writeTestReport(ctx *context.Context) {
	var report *reporter.TestReport
	if r.reportConfig.JSON.Filename != "" {
		report = r.generateTestReport(ctx, report)
		f, err := os.Create(filepath.Join(r.rootDir, r.reportConfig.JSON.Filename))
		if err != nil {
			ctx.Reporter().Fatalf("failed to write JSON test report: %s", err)
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			ctx.Reporter().Fatalf("failed to write JSON test report: %s", err)
		}
	}
	if r.reportConfig.JUnit.Filename != "" {
		report = r.generateTestReport(ctx, report)
		f, err := os.Create(filepath.Join(r.rootDir, r.reportConfig.JUnit.Filename))
		if err != nil {
			ctx.Reporter().Fatalf("failed to write JUnit test report: %s", err)
		}
		defer f.Close()
		enc := xml.NewEncoder(f)
		enc.Indent("", "  ")
		if err := enc.Encode(report); err != nil {
			ctx.Reporter().Fatalf("failed to write JUnit test report: %s", err)
		}
	}
}

func (r *Runner) generateTestReport(ctx *context.Context, report *reporter.TestReport) *reporter.TestReport {
	if report != nil {
		return report
	}
	report, err := reporter.GenerateTestReport(ctx.Reporter())
	if err != nil {
		ctx.Reporter().Fatalf("failed to generate test report: %s", err)
	}
	return report
}
