package template

import (
	"context"
	"sync"

	"github.com/zoncoen/query-go"
	yamlextractor "github.com/zoncoen/query-go/extractor/yaml"

	"github.com/zoncoen/scenarigo/errors"
)

// Lazy represents a value with lazy initialization.
// You can create a Lazy value by using $.
type Lazy func(any) (any, error)

func (t *Template) executeLazyTemplate(ctx context.Context, data any) (any, error) {
	wc, done := t.executeTemplate(ctx, data)
	select {
	case result := <-done:
		return result.v, result.err
	case <-wc.blocked():
		// Delay template evaluation because the actual value is required.
		var once sync.Once
		return Lazy(func(v any) (any, error) {
			var c *waitContext
			once.Do(func() {
				// already executing
				c = wc
			})
			if c == nil {
				// re-execution is required from the second time onwards
				c, done = t.executeTemplate(ctx, data)
			}
			if err := c.set(v); err != nil {
				return nil, err
			}
			result := <-done
			return result.v, result.err
		}), nil
	}
}

func (t *Template) executeTemplate(ctx context.Context, data any) (*waitContext, chan templateResult) {
	wc := newWaitContext(ctx, data)
	done := make(chan templateResult)
	go func() {
		v, err := t.execute(ctx, wc)
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
