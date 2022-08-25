package schema

import (
	"context"
	"errors"
	"time"

	"github.com/lestrrat-go/backoff"
)

// RetryPolicy represents a retry policy.
type RetryPolicy struct {
	Constant    *RetryPolicyConstant    `yaml:"constant"`
	Exponential *RetryPolicyExponential `yaml:"exponential"`
}

// Build returns p as backoff.Policy.
// If p is nil, Build returns the policy which never retry.
func (p *RetryPolicy) Build() (backoff.Policy, error) {
	if p != nil {
		if p.Constant != nil && p.Exponential != nil {
			return nil, errors.New("ambiguous retry policy")
		}
		if p.Constant != nil {
			return p.Constant.Build()
		}
		if p.Exponential != nil {
			return p.Exponential.Build()
		}
	}
	return &retryPolicyNever{}, nil
}

// RetryPolicyConstant represents a constant retry policy.
type RetryPolicyConstant struct {
	Interval       *Duration `yaml:"interval"`
	MaxElapsedTime *Duration `yaml:"maxElapsedTime"`
	MaxRetries     *int      `yaml:"maxRetries"`
}

// Build returns p as backoff.Policy.
func (p *RetryPolicyConstant) Build() (backoff.Policy, error) {
	if p.Interval == nil {
		return nil, errors.New("interval must be specified")
	}
	opts := []backoff.Option{}
	if p.MaxElapsedTime != nil {
		opts = append(opts, backoff.WithMaxElapsedTime(time.Duration(*p.MaxElapsedTime)))
	}
	if p.MaxRetries != nil {
		opts = append(opts, backoff.WithMaxRetries(*p.MaxRetries))
	}
	return backoff.NewConstant(time.Duration(*p.Interval), opts...), nil
}

// RetryPolicyExponential represents a exponential retry policy.
type RetryPolicyExponential struct {
	InitialInterval *Duration `yaml:"initialInterval"`
	Factor          *float64  `yaml:"factor"`
	JitterFactor    *float64  `yaml:"jitterFactor"`
	MaxInterval     *Duration `yaml:"maxInterval"`
	MaxElapsedTime  *Duration `yaml:"maxElapsedTime"`
	MaxRetries      *int      `yaml:"maxRetries"`
}

// Build returns p as backoff.Policy.
func (p *RetryPolicyExponential) Build() (backoff.Policy, error) {
	opts := []backoff.Option{}
	if p.InitialInterval != nil {
		opts = append(opts, backoff.WithInterval(time.Duration(*p.InitialInterval)))
	}
	if p.Factor != nil {
		opts = append(opts, backoff.WithFactor(*p.Factor))
	}
	if p.JitterFactor != nil {
		opts = append(opts, backoff.WithJitterFactor(*p.JitterFactor))
	}
	if p.MaxInterval != nil {
		opts = append(opts, backoff.WithMaxInterval(time.Duration(*p.MaxInterval)))
	}
	if p.MaxElapsedTime != nil {
		opts = append(opts, backoff.WithMaxElapsedTime(time.Duration(*p.MaxElapsedTime)))
	}
	if p.MaxRetries != nil {
		opts = append(opts, backoff.WithMaxRetries(*p.MaxRetries))
	}
	return backoff.NewExponential(opts...), nil
}

type retryPolicyNever struct{}

// Start implements backoff.Policy interface.
func (*retryPolicyNever) Start(ctx context.Context) (backoff.Backoff, backoff.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	b := &retryBackoffNever{
		ctx:    ctx,
		cancel: cancel,
		next:   make(chan struct{}),
	}
	go func() {
		b.next <- struct{}{}
		cancel()
	}()
	return b, backoff.CancelFunc(func() {
		cancel()
	})
}

type retryBackoffNever struct {
	ctx    context.Context
	cancel func()
	next   chan struct{}
}

// Done implements backoff.Backoff interface.
func (b *retryBackoffNever) Done() <-chan struct{} {
	return b.ctx.Done()
}

// Next implements backoff.Backoff interface.
func (b *retryBackoffNever) Next() <-chan struct{} {
	return b.next
}
