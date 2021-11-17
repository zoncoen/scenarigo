package scenarigo

import (
	"fmt"
	"path/filepath"
	"plugin"
	"reflect"
	"sync"

	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/errors"
	"github.com/zoncoen/scenarigo/schema"
)

var (
	plgMu      sync.Mutex
	pluginOpen = func(path string) (lookupper, error) {
		return plugin.Open(path)
	}
)

// loadPlugin loads the plugin safely.
// plugin.Open's documentation says 'this is safe for concurrent use by multiple goroutines' ( https://golang.org/pkg/plugin/#Open )
// BUT we encountered `recursive call during initialization - linker skew` error when loading multiple plugins concurrently.
func loadPlugin(path string) (*plug, error) {
	// TODO: It is not yet known if this process is accurate. If you find a better way, you need to fix this process.
	// see related PR: https://github.com/zoncoen/scenarigo/pull/78
	plgMu.Lock()
	defer plgMu.Unlock()
	p, err := pluginOpen(path)
	if err != nil {
		return nil, err
	}
	return &plug{p}, nil
}

// RunScenario runs a test scenario s.
func RunScenario(ctx *context.Context, s *schema.Scenario) *context.Context {
	ctx = ctx.WithScenarioFilepath(s.Filepath())
	if s.Plugins != nil {
		plugs := map[string]interface{}{}
		for name, path := range s.Plugins {
			path := path
			if root := ctx.PluginDir(); root != "" {
				path = filepath.Join(root, path)
			}
			p, err := loadPlugin(path)
			if err != nil {
				ctx.Reporter().Fatalf("failed to open plugin: %s", err)
			}
			plugs[name] = p
		}
		ctx = ctx.WithPlugins(plugs)
	}

	if s.Vars != nil {
		vars, err := ctx.ExecuteTemplate(s.Vars)
		if err != nil {
			ctx.Reporter().Fatalf("invalid vars: %s", err)
		}
		ctx = ctx.WithVars(vars)
	}

	scnCtx := ctx
	var failed bool
	for idx, step := range s.Steps {
		step := step
		ok := scnCtx.Run(step.Title, func(ctx *context.Context) {
			// following steps are skipped if the previous step failed
			if failed {
				ctx.Reporter().SkipNow()
			}

			ctx = runStep(ctx, s, step, idx)

			// bind values to the scenario context for enable to access from following steps
			if step.Bind.Vars != nil {
				vars, err := ctx.ExecuteTemplate(step.Bind.Vars)
				if err != nil {
					ctx.Reporter().Fatal(
						errors.WithNodeAndColored(
							errors.WrapPath(
								err,
								fmt.Sprintf("steps[%d].bind.vars", idx),
								"invalid bind",
							),
							ctx.Node(),
							ctx.EnabledColor(),
						),
					)
				}
				scnCtx = scnCtx.WithVars(vars)
			}
		})
		if !failed {
			failed = !ok
		}
	}

	return scnCtx
}

// lookupper is an interface wrapper around *plugin.Plugin.
// NOTE: If we use plugin.Plugin in tests, go test with -race flag will fail.
type lookupper interface {
	Lookup(string) (plugin.Symbol, error)
}

type plug struct {
	lookupper
}

// ExtractByKey implements query.KeyExtractor interface.
func (p *plug) ExtractByKey(key string) (interface{}, bool) {
	sym, err := p.Lookup(key)
	if err != nil {
		return nil, false
	}
	// If sym is a pointer to a variable, return the actual variable for convenience.
	if v := reflect.ValueOf(sym); v.Kind() == reflect.Ptr {
		return v.Elem().Interface(), true
	}
	return sym, true
}
