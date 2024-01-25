package grpc

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/goccy/go-yaml"

	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/errors"
	"github.com/zoncoen/scenarigo/internal/queryutil"
	"github.com/zoncoen/scenarigo/internal/reflectutil"
)

// Request represents a request.
type Request struct {
	Client   string      `yaml:"client,omitempty"`
	Method   string      `yaml:"method"`
	Metadata interface{} `yaml:"metadata,omitempty"`
	Message  interface{} `yaml:"message,omitempty"`

	// for backward compatibility
	Body interface{} `yaml:"body,omitempty"`
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
	Status  responseStatus  `yaml:"status,omitempty"`
	Header  *mdMarshaler    `yaml:"header,omitempty"`
	Trailer *mdMarshaler    `yaml:"trailer,omitempty"`
	Message interface{}     `yaml:"message,omitempty"`
	rvalues []reflect.Value `yaml:"-"`
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

func newMDMarshaler(md metadata.MD) *mdMarshaler { return (*mdMarshaler)(&md) }

type mdMarshaler metadata.MD

func (m *mdMarshaler) MarshalYAML() ([]byte, error) {
	mp := make(metadata.MD, len(*m))
	for k, vs := range *m {
		vs := vs
		if !strings.HasSuffix(k, "-bin") {
			mp[k] = vs
			continue
		}
		s := make([]string, len(vs))
		for i, v := range vs {
			v := v
			if !utf8.ValidString(v) {
				v = hex.EncodeToString([]byte(v))
			}
			s[i] = v
		}
		mp[k] = s
	}
	return yaml.Marshal(mp)
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
	if r.Client == "" {
		return ctx, nil, errors.New("gRPC client must be specified")
	}

	x, err := ctx.ExecuteTemplate(r.Client)
	if err != nil {
		return ctx, nil, errors.WrapPath(err, "client", "failed to get client")
	}

	client := reflect.ValueOf(x)
	var method reflect.Value
	for {
		if !client.IsValid() {
			return nil, nil, errors.ErrorPathf("client", "client %s is invalid", r.Client)
		}
		method = client.MethodByName(r.Method)
		if method.IsValid() {
			// method found
			break
		}
		switch client.Kind() {
		case reflect.Interface, reflect.Ptr:
			client = client.Elem()
		default:
			return nil, nil, errors.ErrorPathf("method", "method %s.%s not found", r.Client, r.Method)
		}
	}

	if err := validateMethod(method); err != nil {
		return ctx, nil, errors.ErrorPathf("method", `"%s.%s" must be "func(context.Context, proto.Message, ...grpc.CallOption) (proto.Message, error): %s"`, r.Client, r.Method, err)
	}

	return invoke(ctx, method, r)
}

func validateMethod(method reflect.Value) error {
	if !method.IsValid() {
		return errors.New("invalid")
	}
	if method.Kind() != reflect.Func {
		return errors.New("not function")
	}
	if method.IsNil() {
		return errors.New("method is nil")
	}

	mt := method.Type()
	if n := mt.NumIn(); n != 3 {
		return errors.Errorf("number of arguments must be 3 but got %d", n)
	}
	if t := mt.In(0); !t.Implements(typeContext) {
		return errors.Errorf("first argument must be context.Context but got %s", t.String())
	}
	if t := mt.In(1); !t.Implements(typeMessage) {
		return errors.Errorf("second argument must be proto.Message but got %s", t.String())
	}
	if t := mt.In(2); t != typeCallOpts {
		return errors.Errorf("third argument must be []grpc.CallOption but got %s", t.String())
	}
	if n := mt.NumOut(); n != 2 {
		return errors.Errorf("number of return values must be 2 but got %d", n)
	}
	if t := mt.Out(0); !t.Implements(typeMessage) {
		return errors.Errorf("first return value must be proto.Message but got %s", t.String())
	}
	if t := mt.Out(1); !t.Implements(reflectutil.TypeError) {
		return errors.Errorf("second return value must be error but got %s", t.String())
	}

	return nil
}

func invoke(ctx *context.Context, method reflect.Value, r *Request) (*context.Context, interface{}, error) {
	reqCtx := ctx.RequestContext()
	if r.Metadata != nil {
		x, err := ctx.ExecuteTemplate(r.Metadata)
		if err != nil {
			return ctx, nil, errors.WrapPathf(err, "metadata", "failed to set metadata")
		}
		md, err := reflectutil.ConvertStringsMap(reflect.ValueOf(x))
		if err != nil {
			return nil, nil, errors.WrapPathf(err, "metadata", "failed to set metadata")
		}

		pairs := []string{}
		for k, vs := range md {
			vs := vs
			for _, v := range vs {
				pairs = append(pairs, k, v)
			}
		}
		reqCtx = metadata.AppendToOutgoingContext(reqCtx, pairs...)
	}

	var in []reflect.Value
	for i := 0; i < method.Type().NumIn(); i++ {
		switch i {
		case 0:
			in = append(in, reflect.ValueOf(reqCtx))
		case 1:
			req := reflect.New(method.Type().In(i).Elem()).Interface()
			if err := buildRequestMsg(ctx, req, r.Message); err != nil {
				return ctx, nil, errors.WrapPathf(err, "message", "failed to build request message")
			}

			//nolint:exhaustruct
			dumpReq := &Request{
				Method:  r.Method,
				Message: req,
			}
			reqMD, _ := metadata.FromOutgoingContext(reqCtx)
			if len(reqMD) > 0 {
				dumpReq.Metadata = newMDMarshaler(reqMD)
			}
			ctx = ctx.WithRequest((*RequestExtractor)(dumpReq))
			if b, err := yaml.Marshal(dumpReq); err == nil {
				ctx.Reporter().Logf("request:\n%s", r.addIndent(string(b), indentNum))
			} else {
				ctx.Reporter().Logf("failed to dump request:\n%s", err)
			}

			in = append(in, reflect.ValueOf(req))
		}
	}

	var header, trailer metadata.MD
	in = append(in,
		reflect.ValueOf(grpc.Header(&header)),
		reflect.ValueOf(grpc.Trailer(&trailer)),
	)

	rvalues := method.Call(in)
	message := rvalues[0].Interface()
	var err error
	if rvalues[1].IsValid() && rvalues[1].CanInterface() {
		e, ok := rvalues[1].Interface().(error)
		if ok {
			err = e
		}
	}
	resp := response{
		Status: responseStatus{
			Code:    codes.OK.String(),
			Message: "",
			Details: nil,
		},
		Message: message,
		rvalues: rvalues,
	}
	if len(header) > 0 {
		resp.Header = newMDMarshaler(header)
	}
	if len(trailer) > 0 {
		resp.Trailer = newMDMarshaler(trailer)
	}
	if err != nil {
		if sts, ok := status.FromError(err); ok {
			resp.Status.Code = sts.Code().String()
			resp.Status.Message = sts.Message()
			details := sts.Details()
			if l := len(details); l > 0 {
				m := make(yaml.MapSlice, l)
				for i, d := range details {
					item := yaml.MapItem{
						Key:   "",
						Value: d,
					}
					if msg, ok := d.(proto.Message); ok {
						item.Key = string(proto.MessageName(msg))
					} else {
						item.Key = fmt.Sprintf("%T (not proto.Message)", d)
					}
					m[i] = item
				}
				resp.Status.Details = m
			}
		}
	}
	ctx = ctx.WithResponse((*ResponseExtractor)(&resp))
	if b, err := yaml.Marshal(resp); err == nil {
		ctx.Reporter().Logf("response:\n%s", r.addIndent(string(b), indentNum))
	} else {
		ctx.Reporter().Logf("failed to dump response:\n%s", err)
	}

	return ctx, resp, nil
}

func buildRequestMsg(ctx *context.Context, req interface{}, src interface{}) error {
	x, err := ctx.ExecuteTemplate(src)
	if err != nil {
		return err
	}
	if x == nil {
		return nil
	}
	var buf bytes.Buffer
	if err := yaml.NewEncoder(&buf, yaml.JSON()).Encode(x); err != nil {
		return err
	}
	message, ok := req.(proto.Message)
	if ok {
		if err := protojson.Unmarshal(buf.Bytes(), message); err != nil {
			return err
		}
	}
	return nil
}
