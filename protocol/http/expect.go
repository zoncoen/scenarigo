package http

import (
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/zoncoen/scenarigo/assert"
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/errors"
	"github.com/zoncoen/scenarigo/internal/maputil"
)

// Expect represents expected response values.
type Expect struct {
	Code   string        `yaml:"code"`
	Header yaml.MapSlice `yaml:"header"`
	Body   interface{}   `yaml:"body"`
}

// Build implements protocol.AssertionBuilder interface.
func (e *Expect) Build(ctx *context.Context) (assert.Assertion, error) {
	expectCode := "200"
	if e.Code != "" {
		expectCode = e.Code
	}
	executedCode, err := ctx.ExecuteTemplate(expectCode)
	if err != nil {
		return nil, errors.WrapPathf(err, "code", "invalid expect response: %s", err)
	}
	codeAssertion := assert.Build(executedCode)

	expectBody, err := ctx.ExecuteTemplate(e.Body)
	if err != nil {
		return nil, errors.WrapPathf(err, "body", "invalid expect response")
	}
	assertion := assert.Build(expectBody)

	return assert.AssertionFunc(func(v interface{}) error {
		res, ok := v.(response)
		if !ok {
			return errors.Errorf("expected response but got %T", v)
		}
		if err := assertCode(codeAssertion, res.status); err != nil {
			return errors.WithPath(err, "code")
		}
		if err := e.assertHeader(res.Header); err != nil {
			return errors.WithPath(err, "header")
		}
		if err := assertion.Assert(res.Body); err != nil {
			return errors.WithPath(err, "body")
		}
		return nil
	}), nil
}

func (e *Expect) assertHeader(header map[string][]string) error {
	if len(e.Header) == 0 {
		return nil
	}
	headerMap, err := maputil.ConvertStringsMapSlice(e.Header)
	if err != nil {
		return err
	}
	if err := assert.Build(headerMap).Assert(header); err != nil {
		return err
	}
	return nil
}

func assertCode(assertion assert.Assertion, status string) error {
	strs := strings.SplitN(status, " ", 2)
	if len(strs) != 2 {
		return errors.Errorf(`unexpected response status string: "%s"`, status)
	}
	if err := assertion.Assert(strs[0]); err == nil {
		return nil
	}
	err := assertion.Assert(strs[1])
	if err == nil {
		return nil
	}
	return err
}
