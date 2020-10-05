package assert

import (
	"github.com/zoncoen/query-go"

	"github.com/zoncoen/scenarigo/errors"
)

// Or returns new assertion to ensure that value passes at least one of assertions.
// If the assertions are empty, it returns an error.
func Or(assertions ...Assertion) func(*query.Query) Assertion {
	return func(q *query.Query) Assertion {
		return assertFunc(q, func(v interface{}) error {
			if len(assertions) == 0 {
				return errors.New("empty assertion list")
			}
			errs := []error{}
			for _, assertion := range assertions {
				assertion := assertion
				err := assertion.Assert(v)
				if err == nil {
					return nil
				}
				errs = append(errs, err)
			}
			return errors.Wrap(errors.Errors(errs...), "all assertions failed")
		})
	}
}
