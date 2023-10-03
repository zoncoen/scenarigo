// Package assert provides value assertions.
package assert

import (
	"context"
	"fmt"
	"sync"

	"github.com/goccy/go-yaml"
	"github.com/zoncoen/query-go"
	yamlextractor "github.com/zoncoen/query-go/extractor/yaml"

	"github.com/zoncoen/scenarigo/errors"
	"github.com/zoncoen/scenarigo/template"
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

type buildOpt struct {
	tmplData any
	eqs      []Equaler
}

// BuildOpt represents an option for Build().
type BuildOpt func(*buildOpt)

// FromTemplate is a build option that executes templates before building assertions.
func FromTemplate(data any) BuildOpt {
	return func(opt *buildOpt) {
		opt.tmplData = data
	}
}

// WithEqualers is a build option that enables custom equalers.
func WithEqualers(eqs ...Equaler) BuildOpt {
	return func(opt *buildOpt) {
		opt.eqs = append(opt.eqs, eqs...)
	}
}

// Build builds an assertion from Go value.
// If the Assert method of built assertion isn't called, the context value should be canceled to avoid a goroutine leak.
func Build(ctx context.Context, expect any, fs ...BuildOpt) (Assertion, error) {
	var opt buildOpt
	for _, f := range fs {
		f(&opt)
	}
	var assertions []Assertion
	if expect != nil {
		var err error
		assertions, err = build(ctx, query.New(
			query.ExtractByStructTag("yaml", "json"),
			query.CustomExtractFunc(yamlextractor.MapSliceExtractFunc(false)),
		), expect, &opt)
		if err != nil {
			return nil, fmt.Errorf("failed to build assertion: %w", err)
		}
	}
	return AssertionFunc(func(v interface{}) error {
		errs := []error{}
		for _, assertion := range assertions {
			assertion := assertion
			if err := assertion.Assert(v); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			if len(errs) == 1 {
				return errs[0]
			}
			return errors.Errors(errs...)
		}
		return nil
	}), nil
}

// MustBuild builds an assertion from Go value.
// If the Assert method of built assertion isn't called, the context value should be canceled to avoid a goroutine leak.
// If it fails to build, creates an assertion function that returns the build error.
func MustBuild(ctx context.Context, expect any, fs ...BuildOpt) Assertion {
	assertion, err := Build(ctx, expect, fs...)
	if err != nil {
		return AssertionFunc(func(_ any) error {
			return err
		})
	}
	return assertion
}

func build(ctx context.Context, q *query.Query, expect any, opt *buildOpt) ([]Assertion, error) {
	var assertions []Assertion
	switch v := expect.(type) {
	case yaml.MapSlice:
		for _, item := range v {
			item := item
			k, err := template.Execute(item.Key, opt.tmplData)
			if err != nil {
				return nil, err
			}
			key := fmt.Sprintf("%s", k)
			as, err := build(ctx, q.Key(key), item.Value, opt)
			if err != nil {
				return nil, err
			}
			assertions = append(assertions, as...)
		}
	case []interface{}:
		for i, elm := range v {
			elm := elm
			as, err := build(ctx, q.Index(i), elm, opt)
			if err != nil {
				return nil, err
			}
			assertions = append(assertions, as...)
		}
	default:
		switch v := expect.(type) {
		case string:
			return buildAssertion(ctx, q, v, opt)
		case Assertion:
			assertions = append(assertions, AssertionFunc(func(val interface{}) error {
				got, err := q.Extract(val)
				if err != nil {
					return err
				}
				if err := v.Assert(got); err != nil {
					return errors.WithQuery(err, q)
				}
				return nil
			}))
		case func(*query.Query) Assertion:
			assertions = append(assertions, v(q))
		default:
			as, err := build(ctx, q, Equal(v, opt.eqs...), opt)
			if err != nil {
				return nil, err
			}
			assertions = append(assertions, as...)
		}
	}
	return assertions, nil
}

func buildAssertion(ctx context.Context, q *query.Query, expect any, opt *buildOpt) ([]Assertion, error) {
	wc, done := executeTemplate(ctx, expect, opt.tmplData)

	select {
	case result := <-done:
		if result.err != nil {
			return nil, result.err
		}
		if s, ok := result.v.(string); ok {
			result.v = Equal(s, opt.eqs...)
		}
		return build(ctx, q, result.v, opt)
	case <-wc.blocked():
		// Delay template evaluation because the actual value is required.
		var once sync.Once
		a := AssertionFunc(func(val interface{}) error {
			var c *waitContext
			once.Do(func() {
				// already executing
				c = wc
			})
			if c == nil {
				// re-execution is required from the second time onwards
				c, done = executeTemplate(ctx, expect, opt.tmplData)
			}

			if err := c.set(val); err != nil {
				return err
			}
			result := <-done
			if result.err != nil {
				return result.err
			}
			if pass, err := convert(result.v, false); err == nil {
				if pass {
					return nil
				}
				return errors.New("assertion error")
			}
			return fmt.Errorf("assertion result must be a boolean value but got %T", result.v)
		})
		return build(ctx, q, a, opt)
	}
}

func executeTemplate(ctx context.Context, tmpl any, data any) (*waitContext, chan templateResult) {
	wc := newWaitContext(ctx, data)
	done := make(chan templateResult)
	go func() {
		v, err := template.Execute(tmpl, wc)
		done <- templateResult{
			v:   v,
			err: err,
		}
	}()
	return wc, done
}

type templateResult struct {
	v   any
	err error
}

type waitContext struct {
	any                // base data
	extractActualValue func() (any, bool)
	ready              chan any
	blocked            func() <-chan struct{}
	setOnce            sync.Once
}

func newWaitContext(ctx context.Context, base any) *waitContext {
	block, cancel := context.WithCancel(context.Background())
	ready := make(chan any, 1)
	//nolint:exhaustruct
	return &waitContext{
		any: base,
		extractActualValue: onceValues(func() (any, bool) {
			cancel()
			select {
			case v := <-ready:
				return v, true
			case <-ctx.Done():
				return nil, false
			}
		}),
		ready:   ready,
		blocked: block.Done, //nolint:contextcheck
	}
}

func (c *waitContext) set(v any) error {
	var first bool
	c.setOnce.Do(func() {
		first = true
		c.ready <- v
	})
	if first {
		return nil
	}
	return errors.New("set an actual value twice")
}

// ExtractByKey implements query.KeyExtractor interface.
func (c *waitContext) ExtractByKey(key string) (any, bool) {
	if key == "$" {
		return c.extractActualValue()
	}
	k := query.New(
		query.ExtractByStructTag("yaml", "json"),
		query.CustomExtractFunc(yamlextractor.MapSliceExtractFunc(false)),
	).Key(key)
	res, err := k.Extract(c.any)
	if err != nil {
		return nil, false
	}
	return res, true
}
