package context

import (
	"context"
)

const (
	nameContext = "ctx"
	nameVars    = "vars"
)

var (
	keyContext = struct{}{}
	keyVars    = struct{}{}
)

// WithVars returns a copy of c with v.
func (c *Context) WithVars(v interface{}) *Context {
	if v == nil {
		return c
	}
	vars, _ := c.ctx.Value(keyVars).(Vars)
	vars = vars.Append(v)
	return newContext(
		context.WithValue(c.ctx, keyVars, vars),
		c.reporter,
	)
}

// ExtractByKey implements query.KeyExtractor interface.
func (c *Context) ExtractByKey(key string) (interface{}, bool) {
	switch key {
	case nameContext:
		return c, true
	case nameVars:
		v, ok := c.ctx.Value(keyVars).(Vars)
		if ok {
			return v, true
		}
	}
	return nil, false
}
