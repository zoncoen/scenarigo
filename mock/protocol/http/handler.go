package http

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"github.com/zoncoen/scenarigo/logger"
	"github.com/zoncoen/scenarigo/internal/reflectutil"
	"github.com/zoncoen/scenarigo/mock/protocol"
	httpprotocol "github.com/zoncoen/scenarigo/protocol/http"
	"github.com/zoncoen/scenarigo/protocol/http/marshaler"
)

// NewHandler returns a handler sending mock responses.
func NewHandler(iter *protocol.MockIterator, l logger.Logger) http.Handler {
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
		var resp HTTPResponse
		if mock.Response.Unmarshal(&resp); err != nil {
			writeError(w, err, l)
			return
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
	body, err := marshaler.Get(header.Get("Content-Type")).Marshal(resp.Body)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("failed to marshal response body: %w", err)
	}

	return status, header, body, nil
}
