package grpc

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/errors"
	"github.com/zoncoen/scenarigo/internal/queryutil"
	"github.com/zoncoen/scenarigo/internal/yamlutil"
)

// Request represents a request.
type Request struct {
	Client   string          `yaml:"client,omitempty"`
	Target   string          `yaml:"target,omitempty"`
	Service  string          `yaml:"service,omitempty"`
	Method   string          `yaml:"method"`
	Metadata interface{}     `yaml:"metadata,omitempty"`
	Message  interface{}     `yaml:"message,omitempty"`
	Options  *RequestOptions `yaml:"options,omitempty"`

	// for backward compatibility
	Body interface{} `yaml:"body,omitempty"`
}

// RequestOptions represents request options.
type RequestOptions struct {
	Reflection bool `yaml:"reflection,omitempty"`
}

// RequestExtractor represents a request dump.
type RequestExtractor Request

// ExtractByKey implements query.KeyExtractor interface.
func (r RequestExtractor) ExtractByKey(key string) (interface{}, bool) {
	q := queryutil.New().Key(key)
	if v, err := q.Extract(Request(r)); err == nil {
		return v, true
	}
	// for backward compatibility
	if v, err := q.Extract(r.Message); err == nil {
		return v, true
	}
	return nil, false
}

type response struct {
	Status  responseStatus        `yaml:"status,omitempty"`
	Header  *yamlutil.MDMarshaler `yaml:"header,omitempty"`
	Trailer *yamlutil.MDMarshaler `yaml:"trailer,omitempty"`
	Message interface{}           `yaml:"message,omitempty"`
	rvalues []reflect.Value       `yaml:"-"`
}

type responseStatus struct {
	Code    string        `yaml:"code,omitempty"`
	Message string        `yaml:"message,omitempty"`
	Details yaml.MapSlice `yaml:"details,omitempty"`
}

// ResponseExtractor represents a response dump.
type ResponseExtractor response

// ExtractByKey implements query.KeyExtractor interface.
func (r ResponseExtractor) ExtractByKey(key string) (interface{}, bool) {
	q := queryutil.New().Key(key)
	if v, err := q.Extract(response(r)); err == nil {
		return v, true
	}
	// for backward compatibility
	if v, err := q.Extract(r.Message); err == nil {
		return v, true
	}
	return nil, false
}

const (
	indentNum = 2
)

func (r *Request) addIndent(s string, indentNum int) string {
	indent := strings.Repeat(" ", indentNum)
	lines := []string{}
	for _, line := range strings.Split(s, "\n") {
		if line == "" {
			lines = append(lines, line)
		} else {
			lines = append(lines, fmt.Sprintf("%s%s", indent, line))
		}
	}
	return strings.Join(lines, "\n")
}

// Invoke implements protocol.Invoker interface.
func (r *Request) Invoke(ctx *context.Context) (*context.Context, interface{}, error) {
	opts := &RequestOptions{}
	if r.Options != nil {
		opts = r.Options
	}

	if r.Client == "" {
		return ctx, nil, errors.New("gRPC client must be specified")
	}

	x, err := ctx.ExecuteTemplate(r.Client)
	if err != nil {
		return ctx, nil, errors.WrapPath(err, "client", "failed to get client")
	}
	var client serviceClient = &customServiceClient{
		v: reflect.ValueOf(x),
	}

	return client.invoke(ctx, r, opts)
}

type serviceClient interface {
	invoke(*context.Context, *Request, *RequestOptions) (*context.Context, *response, error)
}
