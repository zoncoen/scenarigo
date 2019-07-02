package scenarigo

import (
	"github.com/k0kubun/pp"
	"github.com/zoncoen/scenarigo/assert"
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/schema"
	"github.com/zoncoen/yaml"
)

func init() {
	pp.ColoringEnabled = false
}

func dumpReqResp(ctx *context.Context) func() {
	return func() {
		if req := ctx.Request(); req != nil {
			if b, err := yaml.Marshal(req); err == nil {
				ctx.Reporter().Logf("request:\n%s", string(b))
			} else {
				ctx.Reporter().Logf("request:\n%s", pp.Sprint(req))
			}
		}
		if resp := ctx.Response(); resp != nil {
			if b, err := yaml.Marshal(resp); err == nil {
				ctx.Reporter().Logf("response:\n%s", string(b))
			} else {
				ctx.Reporter().Logf("response:\n%s", pp.Sprint(resp))
			}
		}
	}
}

func runStep(ctx *context.Context, s *schema.Step, debug *func()) *context.Context {
	// set the function which adds the request and response to log for debugging
	defer func() {
		*debug = dumpReqResp(ctx)
	}()

	if s.Vars != nil {
		vars, err := ctx.ExecuteTemplate(s.Vars)
		if err != nil {
			ctx.Reporter().Fatalf("invalid vars: %s", err)
		}
		ctx = ctx.WithVars(vars)
	}

	if s.Include != "" {
		scenarios, err := schema.LoadScenarios(s.Include)
		if err != nil {
			ctx.Reporter().Fatalf(`failed to include "%s" as step: %s`, s.Include, err)
		}
		if len(scenarios) != 1 {
			ctx.Reporter().Fatalf(`failed to include "%s" as step: must be a scenario`, s.Include)
		}
		return runScenario(ctx, scenarios[0])
	}

	newCtx, resp, err := s.Request.Invoke(ctx)
	if err != nil {
		ctx.Reporter().Fatal(err)
	}
	ctx = newCtx

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
