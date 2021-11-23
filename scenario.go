package scenarigo

import (
	"fmt"
	"path/filepath"

	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/errors"
	"github.com/zoncoen/scenarigo/plugin"
	"github.com/zoncoen/scenarigo/schema"
)

// RunScenario runs a test scenario s.
func RunScenario(ctx *context.Context, s *schema.Scenario) *context.Context {
	ctx = ctx.WithScenarioFilepath(s.Filepath())
	setups := setupMap{}
	if s.Plugins != nil {
		plugs := map[string]interface{}{}
		for name, path := range s.Plugins {
			path := path
			if root := ctx.PluginDir(); root != "" {
				path = filepath.Join(root, path)
			}
			p, err := plugin.Open(path)
			if err != nil {
				ctx.Reporter().Fatalf("failed to open plugin: %s", err)
			}
			plugs[name] = p
			if setup := p.GetSetupEachScenario(); setup != nil {
				setups[name] = setup
			}
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

	ctx, teardown := setups.setup(ctx)
	if ctx.Reporter().Failed() {
		if teardown != nil {
			teardown(ctx)
		}
		return ctx
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

	if teardown != nil {
		teardown(scnCtx)
	}

	return scnCtx
}
