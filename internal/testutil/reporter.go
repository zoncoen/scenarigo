package testutil

import (
	"fmt"
	"sync/atomic"
	"testing"
)

// A Reporter is something that can be used to report test failures.
// It is satisfied by the standard library's *testing.T.
type Reporter interface {
	Fail()
	Failed() bool
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Helper()
}

type reporter struct {
	failed int32
}

// Fail marks the function as having failed but continues execution.
func (r *reporter) Fail() {
	atomic.StoreInt32(&r.failed, 1)
}

// Failed reports whether the function has failed.
func (r *reporter) Failed() bool {
	return atomic.LoadInt32(&r.failed) > 0
}

// Fatal is equivalent to Log followed by FailNow.
func (r *reporter) Fatal(args ...interface{}) {
	r.Fail()
	panic(fmt.Sprint(args...))
}

// Fatalf is equivalent to Logf followed by FailNow.
func (r *reporter) Fatalf(format string, args ...interface{}) {
	r.Fatal(fmt.Sprintf(format, args...))
}

// Helper marks the calling function as a test helper function.
func (r *reporter) Helper() {}

func run(r Reporter, name string, f func(r Reporter)) {
	switch t := r.(type) {
	case *testing.T:
		t.Run(name, func(t *testing.T) {
			f(t)
		})
	default:
		child := &reporter{}
		defer func() {
			err := recover()
			if err != nil {
				fmt.Println(err) //nolint:forbidigo
			}
			if child.Failed() {
				r.Fail()
			}
		}()
		f(child)
	}
}
