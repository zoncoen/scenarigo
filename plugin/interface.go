package plugin

import (
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/schema"
)

// Context represents a scenarigo context.
type Context = context.Context

// Step represents a step plugin.
type Step interface {
	Run(*context.Context, *schema.Step) *context.Context
}

// StepFunc is an adaptor to allow the use of ordinary functions as step.
type StepFunc func(ctx *context.Context, step *schema.Step) *context.Context

// Run implements Step interface.
func (f StepFunc) Run(ctx *context.Context, step *schema.Step) *context.Context {
	return f(ctx, step)
}
