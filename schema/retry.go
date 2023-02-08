package schema

import (
	"context"
	"errors"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// RetryPolicy represents a retry policy.
type RetryPolicy struct {
	Constant    *RetryPolicyConstant    `yaml:"constant"`
	Exponential *RetryPolicyExponential `yaml:"exponential"`
}

// Build returns p as backoff.Policy.
// If p is nil, Build returns the policy which never retry.
func (p *RetryPolicy) Build() (func(ctx context.Context) (context.Context, func()), backoff.BackOff, error) {
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
	return maxElapsedTimeContextFunc(nil), &backoff.StopBackOff{}, nil
}

func maxElapsedTimeContextFunc(t *Duration) func(context.Context) (context.Context, func()) {
	return func(ctx context.Context) (context.Context, func()) {
		if t == nil || *t == 0 {
			return ctx, func() {}
		}
		return context.WithTimeout(ctx, time.Duration(*t))
	}
}

// RetryPolicyConstant represents a constant retry policy.
type RetryPolicyConstant struct {
	Interval       *Duration `yaml:"interval"`       // default value is 1s
	MaxRetries     *int      `yaml:"maxRetries"`     // default value is 5, 0 means forever
	MaxElapsedTime *Duration `yaml:"maxElapsedTime"` // default value is 0, 0 means forever
}

// Build returns p as backoff.Policy.
func (p *RetryPolicyConstant) Build() (func(context.Context) (context.Context, func()), backoff.BackOff, error) {
	interval := time.Second
	if p.Interval != nil {
		interval = time.Duration(*p.Interval)
	}
	var b backoff.BackOff = backoff.NewConstantBackOff(interval)

	maxRetries := 5
	if p.MaxRetries != nil && *p.MaxRetries >= 0 {
		maxRetries = *p.MaxRetries
	}
	if maxRetries > 0 {
		b = backoff.WithMaxRetries(b, uint64(maxRetries))
	}

	return maxElapsedTimeContextFunc(p.MaxElapsedTime), b, nil
}

// RetryPolicyExponential represents a exponential retry policy.
type RetryPolicyExponential struct {
	InitialInterval *Duration `yaml:"initialInterval"` // default value is 500ms
	Factor          *float64  `yaml:"factor"`          // default value is 1.5
	JitterFactor    *float64  `yaml:"jitterFactor"`    // default value is 0.5
	MaxInterval     *Duration `yaml:"maxInterval"`     // default value is 15min
	MaxRetries      *int      `yaml:"maxRetries"`      // default value is 5, 0 means forever
	MaxElapsedTime  *Duration `yaml:"maxElapsedTime"`  // default value is 0, 0 means forever
}

// Build returns p as backoff.Policy.
func (p *RetryPolicyExponential) Build() (func(ctx context.Context) (context.Context, func()), backoff.BackOff, error) {
	eb := backoff.NewExponentialBackOff()
	if p.InitialInterval != nil {
		eb.InitialInterval = time.Duration(*p.InitialInterval)
	}
	if p.Factor != nil {
		eb.Multiplier = *p.Factor
	}
	if p.JitterFactor != nil {
		eb.RandomizationFactor = *p.JitterFactor
	}
	if p.MaxInterval != nil {
		eb.MaxInterval = time.Duration(*p.MaxInterval)
	}
	eb.MaxElapsedTime = 0
	if p.MaxElapsedTime != nil {
		eb.MaxElapsedTime = time.Duration(*p.MaxElapsedTime)
	}

	var b backoff.BackOff = eb
	maxRetries := 5
	if p.MaxRetries != nil && *p.MaxRetries >= 0 {
		maxRetries = *p.MaxRetries
	}
	if maxRetries > 0 {
		b = backoff.WithMaxRetries(b, uint64(maxRetries))
	}

	return maxElapsedTimeContextFunc(p.MaxElapsedTime), b, nil
}
