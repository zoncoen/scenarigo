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
	"github.com/zoncoen/scenarigo/reporter"
	"github.com/zoncoen/scenarigo/schema"

	// Register default protocols.
	_ "github.com/zoncoen/scenarigo/protocol/grpc"
	_ "github.com/zoncoen/scenarigo/protocol/http"
)

// Runner represents a test runner.
type Runner struct {
	pluginDir       *string
	scenarioFiles   []string
	scenarioReaders []io.Reader
	enabledColor    bool
	rootDir         string
	reportConfig    schema.ReportConfig
}

// NewRunner returns a new test runner.
func NewRunner(opts ...func(*Runner) error) (*Runner, error) {
	r := &Runner{}
	r.enabledColor = !color.NoColor
	for _, opt := range opts {
		if err := opt(r); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// WithConfig returns a option which sets configuration.
func WithConfig(config *schema.Config) func(*Runner) error {
	return func(r *Runner) error {
		if config == nil {
			return nil
		}
		var opts []func(r *Runner) error
		opts = append(opts, WithScenarios(config.Scenarios...))
		if config.PluginDirectory != "" {
			opts = append(opts, WithPluginDir(config.PluginDirectory))
		}
		for _, opt := range opts {
			if err := opt(r); err != nil {
				return err
			}
		}
		if config.Output.Colored != nil {
			r.enabledColor = *config.Output.Colored
		}
		r.rootDir = config.Root
		r.reportConfig = config.Output.Report
		return nil
	}
}

// WithScenarios returns a option which finds and sets test scenario files.
func WithScenarios(paths ...string) func(*Runner) error {
	return func(r *Runner) error {
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
// - SCENARIGO_COLOR=(1|true|TRUE)
// nolint:stylecheck
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

// ScenariosFiles returns all scenario file paths.
func (r *Runner) ScenarioFiles() []string {
	return r.scenarioFiles
}

// Scenarios returns map for all scenarios ( key is scenario name and value is steps of scenario ).
func (r *Runner) ScenarioMap(ctx *context.Context, path string) (map[string][]string, error) {
	scns, err := schema.LoadScenarios(path)
	if err != nil {
		return nil, err
	}

	scenarioMap := map[string][]string{}
	ctx.Run(path, func(ctx *context.Context) {
		for _, scn := range scns {
			ctx.Run(scn.Title, func(ctx *context.Context) {
				scenarioName := ctx.Reporter().Name()
				scenarioMap[scenarioName] = []string{}
				for _, step := range scn.Steps {
					ctx.Run(step.Title, func(ctx *context.Context) {
						stepName := ctx.Reporter().Name()
						scenarioMap[scenarioName] = append(scenarioMap[scenarioName], stepName)
					})
				}
			})
		}
	})
	return scenarioMap, nil
}

// Run runs all tests.
func (r *Runner) Run(ctx *context.Context) {
	if r.pluginDir != nil {
		ctx = ctx.WithPluginDir(*r.pluginDir)
	}
	ctx = ctx.WithEnabledColor(r.enabledColor)
	for _, f := range r.scenarioFiles {
		ctx.Run(f, func(ctx *context.Context) {
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
