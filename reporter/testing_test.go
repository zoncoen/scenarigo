package reporter

import (
	"os"
	"testing"
)

func TestFromT(t *testing.T) {
	t.Run("no flag", func(t *testing.T) {
		args := os.Args
		t.Cleanup(func() { os.Args = args })
		os.Args = []string{}
		r := FromT(t).(*reporter)
		if r.context.verbose {
			t.Error("verbose should be false")
		}
		if r.context.matcher != nil {
			t.Fatal("matcher should be nil")
		}
	})
	t.Run("verbose", func(t *testing.T) {
		args := os.Args
		t.Cleanup(func() { os.Args = args })
		os.Args = []string{"-test.v=true"}
		r := FromT(t).(*reporter)
		if !r.context.verbose {
			t.Error("verbose should be true")
		}
	})
	t.Run("matcher", func(t *testing.T) {
		args := os.Args
		t.Cleanup(func() { os.Args = args })
		os.Args = []string{`-test.run=FromT/matcher/bar/^ho`}
		r := FromT(t)
		rptr := r.(*reporter)
		if rptr.context.matcher == nil {
			t.Fatal("matcher should not be nil")
		}

		var (
			called1 bool
			called2 bool
		)
		r.Run("foo", func(r Reporter) {
			r.Run("hoge", func(r Reporter) {
				r.Fatalf("%s should not be executed", r.Name())
			})
			r.Run("fuga", func(r Reporter) {
				r.Fatalf("%s should not be executed", r.Name())
			})
		})
		r.Run("bar", func(r Reporter) {
			r.Run("hoge", func(r Reporter) {
				r.Run("1", func(r Reporter) {
					called1 = true
				})
				r.Run("2", func(r Reporter) {
					called2 = true
				})
			})
			r.Run("fuga", func(r Reporter) {
				r.Fatalf("%s should not be executed", r.Name())
			})
		})
		if !called1 {
			t.Error("TestFromT/bar/hoge/1 should be executed")
		}
		if !called2 {
			t.Error("TestFromT/bar/hoge/2 should be executed")
		}
	})
}

func Test_TestingLog(t *testing.T) {
	rptr := FromT(t)
	rptr.Log("log test")
	rptr.Run("subtest", func(rptr Reporter) {
		rptr.Logf("%s", "log test")
		rptr.Skip()
	})
}
