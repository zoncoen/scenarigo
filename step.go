package scenarigo

import (
	"github.com/zoncoen/scenarigo/assert"
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/schema"
)

func runStep(ctx *context.Context, s *schema.Step) *context.Context {
	var resp interface{}
	var err error
	ctx, resp, err = s.Request.Invoke(ctx)
	if err != nil {
		ctx.Reporter().Fatal(err)
	}

	assertion, err := s.Expect.Build(ctx)
	if err != nil {
		ctx.Reporter().Fatal(err)
	}
	if err := assertion.Assert(resp); err != nil {
		if assertErr, ok := err.(*assert.Error); ok {
			for _, err := range assertErr.Errors {
				ctx.Reporter().Error(err)
			}
			ctx.Reporter().FailNow()
		} else {
			ctx.Reporter().Fatal(err)
		}
	}

	return ctx
}
