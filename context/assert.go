package context

import (
	"github.com/zoncoen/query-go"
	"github.com/zoncoen/scenarigo/assert"
)

var assertions = map[string]func(*query.Query) assert.Assertion{
	"notZero": assert.NotZero,
}
