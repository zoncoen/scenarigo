package scenarigo

import (
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/schema"
)

func runScenario(ctx *context.Context, s *schema.Scenario) *context.Context {
	if s.Vars != nil {
		vars, err := ctx.ExecuteTemplate(s.Vars)
		if err != nil {
			ctx.Reporter().Fatalf("invalid vars: %s", err)
		}
		ctx = ctx.WithVars(vars)
	}

	scnCtx := ctx
	var failed bool
	for _, step := range s.Steps {
		step := step
		ok := scnCtx.Run(step.Title, func(ctx *context.Context) {
			// following steps are skipped if the previous step failed
			if failed {
				ctx.Reporter().SkipNow()
			}

			ctx = runStep(ctx, step)

			// bind values to the scenario context for enable to access from following steps
			if step.Bind.Vars != nil {
				vars, err := ctx.ExecuteTemplate(step.Bind.Vars)
				if err != nil {
					ctx.Reporter().Fatalf("invalid bind: %s", err)
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
