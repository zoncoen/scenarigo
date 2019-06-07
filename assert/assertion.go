// Package assert provides value assertions.
package assert

import (
	"github.com/zoncoen/query-go"
)

// Assertion implements value assertion.
type Assertion interface {
	Assert(v interface{}) error
}

// AssertionFunc is an adaptor to allow the use of ordinary functions as assertions.
type AssertionFunc func(v interface{}) error

// Assert asserts the v.
func (f AssertionFunc) Assert(v interface{}) error {
	return f(v)
}

func assertFunc(q *query.Query, f func(interface{}) error) Assertion {
	return AssertionFunc(func(v interface{}) error {
		got, err := q.Extract(v)
		if err != nil {
			return err
		}
		return f(got)
	})
}
