package grpc

import (
	"bytes"
	"reflect"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	yamljson "github.com/kubernetes-sigs/yaml"
	"github.com/pkg/errors"
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/internal/reflectutil"
	"github.com/zoncoen/yaml"
	"google.golang.org/grpc/metadata"
)

// Request represents a request.
type Request struct {
	Client   string      `yaml:"client"`
	Method   string      `yaml:"method"`
	Metadata interface{} `yaml:"metadata"`
	Body     interface{} `yaml:"body"`
}

// Invoke implements protocol.Invoker interface.
func (r *Request) Invoke(ctx *context.Context) (*context.Context, interface{}, error) {
	if r.Client == "" {
		return ctx, nil, errors.New("gRPC client must be specified")
	}

	x, err := ctx.ExecuteTemplate(r.Client)
	if err != nil {
		return ctx, nil, errors.Errorf("failed to get client: %s", err)
	}

	client := reflect.ValueOf(x)
	var method reflect.Value
	for {
		if !client.IsValid() {
			return nil, nil, errors.Errorf("client %s is invalid", r.Client)
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
			return nil, nil, errors.Errorf("method %s.%s not found", r.Client, r.Method)
		}
	}

	if err := validateMethod(method); err != nil {
		return ctx, nil, errors.Errorf(`"%s.%s" must be "func(context.Context, proto.Message, ...grpc.CallOption) (proto.Message, error): %s"`, r.Client, r.Method, err)
	}

	reqCtx := ctx.RequestContext()
	if r.Metadata != nil {
		x, err := ctx.ExecuteTemplate(r.Metadata)
		if err != nil {
			return ctx, nil, errors.Errorf("failed to set metadata: %s", err)
		}
		md, err := reflectutil.ConvertStringsMap(reflect.ValueOf(x))
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to set metadata")
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
			if err := buildRequestBody(ctx, req, r.Body); err != nil {
				return ctx, nil, errors.Errorf("failed to build request body: %s", err)
			}
			ctx = ctx.WithRequest(req)
			in = append(in, reflect.ValueOf(req))
		}
	}

	vs := method.Call(in)
	ctx = ctx.WithResponse(vs[0].Interface())

	return ctx, vs, nil
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
	if t := mt.Out(1); !t.Implements(typeError) {
		return errors.Errorf("second return value must be error but got %s", t.String())
	}

	return nil
}

func buildRequestBody(ctx *context.Context, req interface{}, src interface{}) error {
	x, err := ctx.ExecuteTemplate(src)
	if err != nil {
		return err
	}
	b, err := yaml.Marshal(x)
	if err != nil {
		return err
	}
	jb, err := yamljson.YAMLToJSON(b)
	if err != nil {
		return err
	}
	message, ok := req.(proto.Message)
	if ok {
		r := bytes.NewReader(jb)
		if err := jsonpb.Unmarshal(r, message); err != nil {
			return err
		}
	}
	return nil
}
