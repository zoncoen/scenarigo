package context

const (
	nameContext  = "ctx"
	nameVars     = "vars"
	nameRequest  = "request"
	nameResponse = "response"
	nameEnv      = "env"
)

// ExtractByKey implements query.KeyExtractor interface.
func (c *Context) ExtractByKey(key string) (interface{}, bool) {
	switch key {
	case nameContext:
		return c, true
	case nameVars:
		v, ok := c.Vars().(Vars)
		if ok {
			return v, true
		}
	case nameRequest:
		v := c.Request()
		if v == nil {
			return nil, false
		}
		return v, true
	case nameResponse:
		v := c.Response()
		if v == nil {
			return nil, false
		}
		return v, true
	case nameEnv:
		return env, true
	}
	return nil, false
}
