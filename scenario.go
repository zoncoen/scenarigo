package scenarigo

import (
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/schema"
)

func runScenario(ctx *context.Context, s *schema.Scenario) *context.Context {
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
		})
		if !failed {
			failed = !ok
		}
	}

	return scnCtx
}
