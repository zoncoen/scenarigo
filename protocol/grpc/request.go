package grpc

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/golang/protobuf/jsonpb" // nolint:staticcheck
	"github.com/golang/protobuf/proto"  // nolint:staticcheck

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/errors"
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

type response struct {
	Header  metadata.MD     `yaml:"header,omitempty"`
	Trailer metadata.MD     `yaml:"trailer,omitempty"`
	Message interface{}     `yaml:"message,omitempty"`
	rvalues []reflect.Value `yaml:"-"`
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

			ctx = ctx.WithRequest(req)
			reqMD, _ := metadata.FromOutgoingContext(reqCtx)
			// nolint:exhaustruct
			if b, err := yaml.Marshal(Request{
				Method:   r.Method,
				Metadata: reqMD,
				Message:  req,
			}); err == nil {
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
	resp := response{
		Header:  header,
		Trailer: trailer,
		Message: message,
		rvalues: rvalues,
	}
	ctx = ctx.WithResponse(message)
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
	var buf bytes.Buffer
	if err := yaml.NewEncoder(&buf, yaml.JSON()).Encode(x); err != nil {
		return err
	}
	message, ok := req.(proto.Message)
	if ok {
		r := bytes.NewReader(buf.Bytes())
		if err := jsonpb.Unmarshal(r, message); err != nil {
			return err
		}
	}
	return nil
}
