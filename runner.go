package scenarigo

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/schema"

	// register default protocols
	_ "github.com/zoncoen/scenarigo/protocol/grpc"
	_ "github.com/zoncoen/scenarigo/protocol/http"
)

// Runner represents a test runner.
type Runner struct {
	scenarioFiles []string
}

// NewRunner returns a new test runner.
func NewRunner(opts ...func(*Runner) error) (*Runner, error) {
	r := &Runner{}
	for _, opt := range opts {
		if err := opt(r); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// WithScenarios returns a option which finds and sets test scenario files.
func WithScenarios(paths ...string) func(*Runner) error {
	return func(r *Runner) error {
		files, err := getAllFiles(paths...)
		if err != nil {
			return err
		}
		r.scenarioFiles = files
		return nil
	}
}

func getAllFiles(paths ...string) ([]string, error) {
	files := []string{}
	for _, path := range paths {
		p, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if p.IsDir() {
			fis, err := ioutil.ReadDir(path)
			if err != nil {
				return nil, err
			}
			for _, fi := range fis {
				fs, err := getAllFiles(filepath.Join(path, fi.Name()))
				if err != nil {
					return nil, err
				}
				files = append(files, fs...)
			}
			continue
		}
		files = append(files, path)
	}
	return files, nil
}

// Run runs all tests.
func (r *Runner) Run(ctx *context.Context) {
	for _, f := range r.scenarioFiles {
		ctx.Run(f, func(ctx *context.Context) {
			scns, err := schema.LoadScenarios(f)
			if err != nil {
				ctx.Reporter().Fatalf("failed to load scenarios: %s", err)
			}
			for _, scn := range scns {
				scn := scn
				ctx.Run(scn.Title, func(ctx *context.Context) {
					ctx.Reporter().Parallel()
					_ = runScenario(ctx, scn)
				})
			}
		})
	}
}
