package context

import "github.com/zoncoen/scenarigo/internal/queryutil"

// Vars represents context variables.
type Vars []interface{}

// Append appends v to context variables.
func (vars Vars) Append(v interface{}) Vars {
	if v == nil {
		return vars
	}
	vars = append(vars, v)
	return vars
}

// ExtractByKey implements query.KeyExtractor interface.
func (vars Vars) ExtractByKey(key string) (interface{}, bool) {
	k := queryutil.New().Key(key)
	for i := len(vars) - 1; i >= 0; i-- {
		if v, err := k.Extract(vars[i]); err == nil {
			return v, true
		}
	}
	return nil, false
}
