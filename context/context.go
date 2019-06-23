// Package context provides the test context of scenarigo.
package context

import (
	"context"
	"testing"

	"github.com/zoncoen/scenarigo/reporter"
)

type (
	keyVars     struct{}
	keyRequest  struct{}
	keyResponse struct{}
)

// Context represents a scenarigo context.
type Context struct {
	ctx      context.Context
	reqCtx   context.Context
	reporter reporter.Reporter
}

// New returns a new scenarigo context.
func New(r reporter.Reporter) *Context {
	return newContext(context.Background(), context.Background(), r)
}

// FromT creates a new context from t.
func FromT(t *testing.T) *Context {
	return newContext(context.Background(), context.Background(), reporter.FromT(t))
}

func newContext(ctx context.Context, reqCtx context.Context, r reporter.Reporter) *Context {
	return &Context{
		ctx:      ctx,
		reqCtx:   reqCtx,
		reporter: r,
	}
}

// WithRequestContext returns the context.Context for request.
func (c *Context) WithRequestContext(reqCtx context.Context) *Context {
	return newContext(
		c.ctx,
		reqCtx,
		c.reporter,
	)
}

// RequestContext returns the context.Context for request.
func (c *Context) RequestContext() context.Context {
	return c.reqCtx
}

// WithReporter returns a copy of c with new test reporter.
func (c *Context) WithReporter(r reporter.Reporter) *Context {
	return newContext(c.ctx, c.reqCtx, r)
}

// Reporter returns the reporter of context.
func (c *Context) Reporter() reporter.Reporter {
	return c.reporter
}

// WithVars returns a copy of c with v.
func (c *Context) WithVars(v interface{}) *Context {
	if v == nil {
		return c
	}
	vars, _ := c.ctx.Value(keyVars{}).(Vars)
	vars = vars.Append(v)
	return newContext(
		context.WithValue(c.ctx, keyVars{}, vars),
		c.reqCtx,
		c.reporter,
	)
}

// Vars returns the context variables.
func (c *Context) Vars() interface{} {
	return c.ctx.Value(keyVars{})
}

// WithRequest returns a copy of c with request.
func (c *Context) WithRequest(req interface{}) *Context {
	if req == nil {
		return c
	}
	return newContext(
		context.WithValue(c.ctx, keyRequest{}, req),
		c.reqCtx,
		c.reporter,
	)
}

// Request returns the request.
func (c *Context) Request() interface{} {
	return c.ctx.Value(keyRequest{})
}

// WithResponse returns a copy of c with response.
func (c *Context) WithResponse(resp interface{}) *Context {
	if resp == nil {
		return c
	}
	return newContext(
		context.WithValue(c.ctx, keyResponse{}, resp),
		c.reqCtx,
		c.reporter,
	)
}

// Response returns the response.
func (c *Context) Response() interface{} {
	return c.ctx.Value(keyResponse{})
}

// Run runs f as a subtest of c called name.
func (c *Context) Run(name string, f func(*Context)) bool {
	return c.Reporter().Run(name, func(r reporter.Reporter) { f(c.WithReporter(r)) })
}
