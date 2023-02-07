package schema

import (
	"context"
	"errors"
	"time"

	"github.com/lestrrat-go/backoff/v2"
)

// RetryPolicy represents a retry policy.
type RetryPolicy struct {
	Constant    *RetryPolicyConstant    `yaml:"constant"`
	Exponential *RetryPolicyExponential `yaml:"exponential"`
}

// Build returns p as backoff.Policy.
// If p is nil, Build returns the policy which never retry.
func (p *RetryPolicy) Build() (func(ctx context.Context) (context.Context, func()), backoff.Policy, error) {
	if p != nil {
		if p.Constant != nil && p.Exponential != nil {
			return nil, nil, errors.New("ambiguous retry policy")
		}
		if p.Constant != nil {
			return p.Constant.Build()
		}
		if p.Exponential != nil {
			return p.Exponential.Build()
		}
	}
	return maxElapsedTimeContextFunc(nil), backoff.NewNull(), nil
}

func maxElapsedTimeContextFunc(t *Duration) func(context.Context) (context.Context, func()) {
	return func(ctx context.Context) (context.Context, func()) {
		if t == nil {
			return ctx, func() {}
		}
		return context.WithTimeout(ctx, time.Duration(*t))
	}
}

// RetryPolicyConstant represents a constant retry policy.
type RetryPolicyConstant struct {
	Interval       *Duration `yaml:"interval"` // default value is 1 min
	MaxElapsedTime *Duration `yaml:"maxElapsedTime"`
	MaxRetries     *int      `yaml:"maxRetries"` // default value is 10 / 0 means forever
}

// Build returns p as backoff.Policy.
func (p *RetryPolicyConstant) Build() (func(context.Context) (context.Context, func()), backoff.Policy, error) {
	opts := []backoff.Option{}
	if p.Interval != nil {
		opts = append(opts, backoff.WithInterval(time.Duration(*p.Interval)))
	}
	if p.MaxRetries != nil {
		opts = append(opts, backoff.WithMaxRetries(*p.MaxRetries))
	}
	return maxElapsedTimeContextFunc(p.MaxElapsedTime), backoff.NewConstantPolicy(opts...), nil
}

// RetryPolicyExponential represents a exponential retry policy.
type RetryPolicyExponential struct {
	InitialInterval *Duration `yaml:"initialInterval"` // default value is 500 ms
	Factor          *float64  `yaml:"factor"`          // default value is 1.5
	JitterFactor    *float64  `yaml:"jitterFactor"`    // must be between 0.0 < v < 1.0
	MaxInterval     *Duration `yaml:"maxInterval"`     // default value is 1 min
	MaxElapsedTime  *Duration `yaml:"maxElapsedTime"`
	MaxRetries      *int      `yaml:"maxRetries"` // default value is 10 / 0 means forever
}

// Build returns p as backoff.Policy.
func (p *RetryPolicyExponential) Build() (func(ctx context.Context) (context.Context, func()), backoff.Policy, error) {
	opts := []backoff.ExponentialOption{}
	if p.InitialInterval != nil {
		opts = append(opts, backoff.WithMinInterval(time.Duration(*p.InitialInterval)))
	}
	if p.Factor != nil {
		opts = append(opts, backoff.WithMultiplier(*p.Factor))
	}
	if p.JitterFactor != nil {
		if f := *p.JitterFactor; f <= 0.0 || 1.0 <= f {
			return nil, nil, errors.New("jitterFactor must be 0.0 < v < 1.0")
		}
		opts = append(opts, backoff.WithJitterFactor(*p.JitterFactor))
	}
	if p.MaxInterval != nil {
		opts = append(opts, backoff.WithMaxInterval(time.Duration(*p.MaxInterval)))
	}
	if p.MaxRetries != nil {
		opts = append(opts, backoff.WithMaxRetries(*p.MaxRetries))
	}
	return maxElapsedTimeContextFunc(p.MaxElapsedTime), backoff.NewExponentialPolicy(opts...), nil
}
