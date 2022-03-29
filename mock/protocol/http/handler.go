package http

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"

	"github.com/goccy/go-yaml"

	"github.com/zoncoen/scenarigo/assert"
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/errors"
	"github.com/zoncoen/scenarigo/internal/assertutil"
	"github.com/zoncoen/scenarigo/internal/reflectutil"
	"github.com/zoncoen/scenarigo/logger"
	"github.com/zoncoen/scenarigo/mock/protocol"
	httpprotocol "github.com/zoncoen/scenarigo/protocol/http"
	"github.com/zoncoen/scenarigo/protocol/http/marshaler"
	"github.com/zoncoen/scenarigo/protocol/http/unmarshaler"
)

// NewHandler returns a handler sending mock responses.
func NewHandler(iter *protocol.MockIterator, l logger.Logger) http.Handler {
	ctx := context.New(nil)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mock, err := iter.Next()
		if err != nil {
			writeError(w, err, l)
			return
		}
		if mock.Protocol != "http" {
			err := fmt.Errorf("received HTTP request but the mock protocol is %q", mock.Protocol)
			writeError(w, err, l)
			return
		}

		var e expect
		if err := mock.Expect.Unmarshal(&e); err != nil {
			writeError(w, fmt.Errorf("failed to unmarshal expect: %w", err), l)
			return
		}
		assertion, err := e.build(ctx)
		if err != nil {
			writeError(w, fmt.Errorf("failed to build assertion: %w", err), l)
			return
		}

		b, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, fmt.Errorf("failed to read request body: %w", err), l)
			return
		}
		var body interface{}
		if len(b) > 0 {
			mt := r.Header.Get("Content-Type")
			if mt == "" {
				mt = "application/json"
				r.Header.Set("Content-Type", mt)
			}
			if err := unmarshaler.Get(mt).Unmarshal(b, &body); err != nil {
				writeError(w, fmt.Errorf("failed to unmarshal request body: %w", err), l)
				return
			}
		}
		newCtx := ctx.WithRequest(map[string]interface{}{
			"header": r.Header,
			"body":   body,
		})

		if err := assertion.Assert(&request{
			path:   r.URL.Path,
			header: r.Header,
			body:   body,
		}); err != nil {
			writeError(w, fmt.Errorf("assertion error: %w", err), l)
			return
		}

		var resp HTTPResponse
		if err := mock.Response.Unmarshal(&resp); err != nil {
			writeError(w, fmt.Errorf("failed to unmarshal response: %w", err), l)
			return
		}

		v, err := newCtx.ExecuteTemplate(resp)
		if err != nil {
			writeError(w, fmt.Errorf("failed to execute template of response body: %w", err), l)
			return
		}
		if r, ok := v.(HTTPResponse); !ok {
			writeError(w, fmt.Errorf("failed to execute template of response body: %w", err), l)
			return
		} else {
			resp = r
		}
		if err := resp.Write(w); err != nil {
			l.Error(err, "failed to write response")
		}
	})
}

func writeError(w http.ResponseWriter, err error, l logger.Logger) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
	l.Error(err, "internal server error")
}

type request struct {
	path   string
	header http.Header
	body   interface{}
}

type expect struct {
	Path   *string       `yaml:"path"`
	Header yaml.MapSlice `yaml:"header"`
	Body   interface{}   `yaml:"body"`
}

func (e *expect) build(ctx *context.Context) (assert.Assertion, error) {
	var pathAssertion assert.Assertion = assert.AssertionFunc(func(_ interface{}) error {
		return nil
	})
	if e.Path != nil {
		expectPath, err := ctx.ExecuteTemplate(*e.Path)
		if err != nil {
			return nil, errors.WrapPathf(err, "path", "invalid expect path")
		}
		pathAssertion = assert.Build(expectPath)
	}

	headerAssertion, err := assertutil.BuildHeaderAssertion(ctx, e.Header)

	expectBody, err := ctx.ExecuteTemplate(e.Body)
	if err != nil {
		return nil, errors.WrapPathf(err, "body", "invalid expect response")
	}
	assertion := assert.Build(expectBody)

	return assert.AssertionFunc(func(v interface{}) error {
		req, ok := v.(*request)
		if !ok {
			return errors.Errorf("expected request but got %T", v)
		}
		if err := pathAssertion.Assert(req.path); err != nil {
			return errors.WithPath(err, "path")
		}
		if err := headerAssertion.Assert(req.header); err != nil {
			return errors.WithPath(err, "header")
		}
		if err := assertion.Assert(req.body); err != nil {
			return errors.WithPath(err, "body")
		}
		return nil
	}), nil
}

// HTTPResponse represents an HTTP response.
type HTTPResponse httpprotocol.Expect

// Write writes header and body.
func (resp *HTTPResponse) Write(w http.ResponseWriter) error {
	status, header, body, err := resp.extract()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return err
	}
	for k, vs := range header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(status)
	_, err = w.Write(body)
	return err
}

func (resp *HTTPResponse) extract() (int, http.Header, []byte, error) {
	status := http.StatusOK
	if resp.Code != "" {
		var err error
		status, err = strconv.Atoi(resp.Code)
		if err != nil {
			return 0, nil, nil, fmt.Errorf("invalid status code %q: %w", resp.Code, err)
		}
	}

	header := make(http.Header, len(resp.Header))
	for _, hdr := range resp.Header {
		k, err := reflectutil.ConvertString(reflect.ValueOf(hdr.Key))
		if err != nil {
			return 0, nil, nil, fmt.Errorf("header key must be a string: %+v is invalid: %w", hdr.Key, err)
		}
		vs, err := reflectutil.ConvertStrings(reflect.ValueOf(hdr.Value))
		if err != nil {
			return 0, nil, nil, fmt.Errorf("invalid header value: %s: %w", k, err)
		}
		for _, v := range vs {
			header.Add(k, v)
		}
	}

	if resp.Body == nil {
		return status, header, nil, nil
	}

	mt := header.Get("Content-Type")
	if mt == "" {
		mt = "application/json"
		header.Set("Content-Type", mt)
	}
	body, err := marshaler.Get(mt).Marshal(resp.Body)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("failed to marshal response body: %w", err)
	}

	return status, header, body, nil
}
