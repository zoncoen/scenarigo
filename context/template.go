package context

import (
	"github.com/zoncoen/scenarigo/template"
)

// ExecuteTemplate executes template strings in context.
func (c *Context) ExecuteTemplate(i interface{}) (interface{}, error) {
	return template.Execute(c.RequestContext(), i, c)
}
