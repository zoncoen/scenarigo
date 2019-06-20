// Package context provides the test context of scenarigo.
package context

import (
	"context"
	"testing"

	"github.com/zoncoen/scenarigo/reporter"
)

// Context represents a scenarigo context.
type Context struct {
	ctx      context.Context
	reporter reporter.Reporter
}

// New returns a new scenarigo context.
func New(r reporter.Reporter) *Context {
	return newContext(context.Background(), r)
}

// FromT creates a new context from t.
func FromT(t *testing.T) *Context {
	return newContext(context.Background(), reporter.FromT(t))
}

func newContext(ctx context.Context, r reporter.Reporter) *Context {
	return &Context{
		ctx:      ctx,
		reporter: r,
	}
}

// WithReporter returns a copy of c with new test reporter.
func (c *Context) WithReporter(r reporter.Reporter) *Context {
	return newContext(c.ctx, r)
}

// Reporter returns the reporter of context.
func (c *Context) Reporter() reporter.Reporter {
	return c.reporter
}

// Run runs f as a subtest of c called name.
func (c *Context) Run(name string, f func(*Context)) bool {
	return c.Reporter().Run(name, func(r reporter.Reporter) { f(c.WithReporter(r)) })
}
