package schema

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lestrrat-go/backoff/v2"
)

func TestRetryPolicyConstant(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		maxElapsedTime := time.Millisecond
		p := &RetryPolicy{
			Constant: &RetryPolicyConstant{
				MaxElapsedTime: (*Duration)(&maxElapsedTime),
			},
		}
		ctxFunc, policy, err := p.Build()
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := ctxFunc(context.Background())
		defer cancel()

		var i int
		b := policy.Start(ctx)
		for backoff.Continue(b) {
			i++
		}
		if got, expect := i, 1; got != expect {
			t.Errorf("expect %d but got %d", expect, got)
		}
	})
	t.Run("interval", func(t *testing.T) {
		maxElapsedTime := time.Second
		interval := time.Millisecond
		p := &RetryPolicy{
			Constant: &RetryPolicyConstant{
				MaxElapsedTime: (*Duration)(&maxElapsedTime),
				Interval:       (*Duration)(&interval),
				// MaxRetries: &maxRetries, // default value is 10
			},
		}
		ctxFunc, policy, err := p.Build()
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := ctxFunc(context.Background())
		defer cancel()

		var i int
		b := policy.Start(ctx)
		for backoff.Continue(b) {
			i++
		}
		if got, expect := i, 11; got != expect {
			t.Errorf("expect %d but got %d", expect, got)
		}
	})
	t.Run("max retires", func(t *testing.T) {
		maxElapsedTime := time.Second
		interval := time.Millisecond
		maxRetries := 1
		p := &RetryPolicy{
			Constant: &RetryPolicyConstant{
				MaxElapsedTime: (*Duration)(&maxElapsedTime),
				Interval:       (*Duration)(&interval),
				MaxRetries:     &maxRetries,
			},
		}
		ctxFunc, policy, err := p.Build()
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := ctxFunc(context.Background())
		defer cancel()

		var i int
		b := policy.Start(ctx)
		for backoff.Continue(b) {
			i++
		}
		if got, expect := i, 2; got != expect {
			t.Errorf("expect %d but got %d", expect, got)
		}
	})
}

func TestRetryPolicyExponential(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		maxElapsedTime := time.Millisecond
		p := &RetryPolicy{
			Exponential: &RetryPolicyExponential{
				MaxElapsedTime: (*Duration)(&maxElapsedTime),
			},
		}
		ctxFunc, policy, err := p.Build()
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := ctxFunc(context.Background())
		defer cancel()

		var i int
		b := policy.Start(ctx)
		for backoff.Continue(b) {
			i++
		}
		if got, expect := i, 1; got != expect {
			t.Errorf("expect %d but got %d", expect, got)
		}
	})
	t.Run("interval", func(t *testing.T) {
		maxElapsedTime := time.Second
		initialInterval := 100 * time.Millisecond
		factor := 2.0
		maxInterval := 300 * time.Millisecond
		p := &RetryPolicy{
			Exponential: &RetryPolicyExponential{
				MaxElapsedTime:  (*Duration)(&maxElapsedTime),
				InitialInterval: (*Duration)(&initialInterval),
				Factor:          &factor,
				MaxInterval:     (*Duration)(&maxInterval),
			},
		}
		ctxFunc, policy, err := p.Build()
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := ctxFunc(context.Background())
		defer cancel()

		var i int
		b := policy.Start(ctx)
		for backoff.Continue(b) {
			i++
		}
		if got, expect := i, 5; got != expect {
			t.Errorf("expect %d but got %d", expect, got)
		}
	})
	t.Run("max retires", func(t *testing.T) {
		maxElapsedTime := time.Second
		initialInterval := 100 * time.Millisecond
		factor := 2.0
		maxInterval := 300 * time.Millisecond
		maxRetries := 2
		p := &RetryPolicy{
			Exponential: &RetryPolicyExponential{
				MaxElapsedTime:  (*Duration)(&maxElapsedTime),
				InitialInterval: (*Duration)(&initialInterval),
				Factor:          &factor,
				MaxInterval:     (*Duration)(&maxInterval),
				MaxRetries:      &maxRetries,
			},
		}
		ctxFunc, policy, err := p.Build()
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := ctxFunc(context.Background())
		defer cancel()

		var i int
		b := policy.Start(ctx)
		for backoff.Continue(b) {
			i++
		}
		if got, expect := i, 3; got != expect {
			t.Errorf("expect %d but got %d", expect, got)
		}
	})
	t.Run("jitter factor", func(t *testing.T) {
		tests := []struct {
			jitterFactor float64
			expectErr    bool
		}{
			{
				jitterFactor: 0.0,
				expectErr:    true,
			},
			{
				jitterFactor: 0.01,
				expectErr:    false,
			},
			{
				jitterFactor: 0.99,
				expectErr:    false,
			},
			{
				jitterFactor: 1.0,
				expectErr:    true,
			},
		}
		for _, test := range tests {
			test := test
			t.Run(fmt.Sprint(test.jitterFactor), func(t *testing.T) {
				p := &RetryPolicy{
					Exponential: &RetryPolicyExponential{
						JitterFactor: &test.jitterFactor,
					},
				}
				_, _, err := p.Build()
				if test.expectErr && err == nil {
					t.Fatal("no error")
				}
				if !test.expectErr && err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
			})
		}
	})
}

func TestRetryPolicyNull(t *testing.T) {
	p := &RetryPolicy{}
	ctxFunc, policy, err := p.Build()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := ctxFunc(context.Background())
	defer cancel()

	var i int
	b := policy.Start(ctx)
	for backoff.Continue(b) {
		i++
	}
	if got, expect := i, 1; got != expect {
		t.Errorf("expect %d but got %d", expect, got)
	}
}
