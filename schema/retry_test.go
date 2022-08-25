package schema

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lestrrat-go/backoff"
)

func TestNeverBackoff(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	policy := &retryPolicyNever{}
	b, backoffCancel := policy.Start(ctx)
	defer backoffCancel()

	done := make(chan error)
	go func() {
		if expect, got := true, backoff.Continue(b); got != expect {
			done <- fmt.Errorf("expect %t but got %t", expect, got)
		}
		if expect, got := false, backoff.Continue(b); got != expect {
			done <- fmt.Errorf("expect %t but got %t", expect, got)
		}
		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

func TestRetryPolicyConstant_Build(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		tests := map[string]struct {
			policy *RetryPolicyConstant
			expect string
		}{
			"no interval": {
				policy: &RetryPolicyConstant{},
				expect: "interval must be specified",
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				if _, err := test.policy.Build(); err == nil {
					t.Fatal("no error")
				} else if got, expect := err.Error(), test.expect; got != expect {
					t.Errorf("expect %q but got %q", expect, got)
				}
			})
		}
	})
}
