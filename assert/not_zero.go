package assert

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/zoncoen/query-go"
)

// NotZero returns an assertion to ensure a value is not zero value.
func NotZero(q *query.Query) Assertion {
	return assertFunc(q, func(v interface{}) error {
		if v == nil || reflect.DeepEqual(v, reflect.Zero(reflect.TypeOf(v)).Interface()) {
			return errors.Errorf("%s: expected not zero value", q.String())
		}
		return nil
	})
}
