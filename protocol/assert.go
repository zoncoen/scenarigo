package protocol

import (
	"fmt"

	"github.com/zoncoen/query-go"
	"github.com/zoncoen/scenarigo/assert"
	"github.com/zoncoen/scenarigo/query/extractor"
	"github.com/zoncoen/yaml"
)

// CreateAssertion is a utility function to create Go value assertion from YAML.
func CreateAssertion(expect interface{}) assert.Assertion {
	var assertions []assert.Assertion
	if expect != nil {
		assertions = createAssertions(query.New(), expect)
	}
	return assert.AssertionFunc(func(v interface{}) error {
		var assertErr error
		for _, assertion := range assertions {
			assertion := assertion
			if err := assertion.Assert(v); err != nil {
				assertErr = assert.AppendError(assertErr, err)
			}
		}
		return assertErr
	})
}

func createAssertions(q *query.Query, expect interface{}) []assert.Assertion {
	var assertions []assert.Assertion
	switch v := expect.(type) {
	case yaml.MapSlice:
		for _, item := range v {
			item := item
			key := fmt.Sprintf("%s", item.Key)
			assertions = append(assertions, createAssertions(q.Append(extractor.Key(key)), item.Value)...)
		}
	case []interface{}:
		for i, elm := range v {
			elm := elm
			assertions = append(assertions, createAssertions(q.Index(i), elm)...)
		}
	default:
		switch v := expect.(type) {
		case func(*query.Query) assert.Assertion:
			assertions = append(assertions, v(q))
		default:
			assertions = append(assertions, assert.Equal(q, v))
		}
	}
	return assertions
}
