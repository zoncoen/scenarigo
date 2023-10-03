package grpc

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/goccy/go-yaml"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	// Register proto messages to unmarshal com.google.protobuf.Any.
	_ "google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/zoncoen/scenarigo/assert"
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/errors"
	"github.com/zoncoen/scenarigo/internal/assertutil"
)

// Expect represents expected response values.
type Expect struct {
	Code    string        `yaml:"code,omitempty"`
	Message interface{}   `yaml:"message,omitempty"`
	Status  ExpectStatus  `yaml:"status,omitempty"`
	Header  yaml.MapSlice `yaml:"header,omitempty"`
	Trailer yaml.MapSlice `yaml:"trailer,omitempty"`

	// for backward compatibility
	Body interface{} `yaml:"body,omitempty"`
}

// ExpectStatus represents expected gRPC status.
type ExpectStatus struct {
	Code    string                     `yaml:"code"`
	Message string                     `yaml:"message"`
	Details []map[string]yaml.MapSlice `yaml:"details"`
}

// Build implements protocol.AssertionBuilder interface.
func (e *Expect) Build(ctx *context.Context) (assert.Assertion, error) {
	codePath := "code"
	expectCode := "OK"
	if e.Code != "" {
		expectCode = e.Code
	}
	if e.Status.Code != "" {
		codePath = "status.code"
		expectCode = e.Status.Code
	}
	codeAssertion, err := assert.Build(ctx.RequestContext(), expectCode, assert.FromTemplate(ctx))
	if err != nil {
		return nil, errors.WrapPathf(err, codePath, "invalid expect status code")
	}

	var statusMsgAssertion assert.Assertion
	if e.Status.Message != "" {
		statusMsgAssertion, err = assert.Build(ctx.RequestContext(), e.Status.Message, assert.FromTemplate(ctx))
		if err != nil {
			return nil, errors.WrapPathf(err, "status.message", "invalid expect status message")
		}
	}

	statusDetailAssertions, err := e.buildStatusDetailAssertions(ctx)
	if err != nil {
		return nil, err
	}

	headerAssertion, err := assertutil.BuildHeaderAssertion(ctx, e.Header)
	if err != nil {
		return nil, errors.WrapPathf(err, "header", "invalid expect header")
	}
	trailerAssertion, err := assertutil.BuildHeaderAssertion(ctx, e.Trailer)
	if err != nil {
		return nil, errors.WrapPathf(err, "trailer", "invalid expect trailer")
	}

	msgAssertion, err := assert.Build(ctx.RequestContext(), e.Message, assert.FromTemplate(ctx))
	if err != nil {
		return nil, errors.WrapPathf(err, "message", "invalid expect response message")
	}

	return assert.AssertionFunc(func(v interface{}) error {
		resp, ok := v.(response)
		if !ok {
			return errors.Errorf(`failed to convert to response type. type is %s`, reflect.TypeOf(v))
		}
		message, stErr, err := extract(resp)
		if err != nil {
			return err
		}
		if err := assertStatusCode(codeAssertion, stErr); err != nil {
			return errors.WithPath(err, codePath)
		}
		if err := e.assertStatusMessage(statusMsgAssertion, stErr); err != nil {
			return errors.WithPath(err, "status.message")
		}
		if err := e.assertStatusDetails(statusDetailAssertions, stErr); err != nil {
			return errors.WithPath(err, "status")
		}
		if err := headerAssertion.Assert(resp.Header); err != nil {
			return errors.WithPath(err, "header")
		}
		if err := trailerAssertion.Assert(resp.Trailer); err != nil {
			return errors.WithPath(err, "trailer")
		}
		if err := msgAssertion.Assert(message); err != nil {
			return errors.WithPath(err, "message")
		}
		return nil
	}), nil
}

func (e *Expect) buildStatusDetailAssertions(ctx *context.Context) ([]assert.Assertion, error) {
	var statusDetailAssertions []assert.Assertion
	if l := len(e.Status.Details); l > 0 {
		statusDetailAssertions = make([]assert.Assertion, l)
		for i, d := range e.Status.Details {
			if len(d) != 1 {
				return nil, errors.ErrorPath(fmt.Sprintf("status.details[%d]", i), "an element of status.details list must be a map of size 1 with the detail message name as the key and the value as the detail message object")
			}
			for k, v := range d {
				fullName, err := assert.Build(ctx.RequestContext(), k, assert.FromTemplate(ctx), assert.WithEqualers(
					assert.EqualerFunc(func(x, y any) (bool, error) {
						if s, ok := x.(string); ok {
							return true, assert.Equal(protoreflect.FullName(s)).Assert(y)
						}
						return false, nil
					}),
				))
				if err != nil {
					return nil, errors.WrapPathf(err, fmt.Sprintf("status.details[%d].'%s'", i, k), "invalid expect status detail message name")
				}

				fields, err := assert.Build(ctx.RequestContext(), v, assert.FromTemplate(ctx))
				if err != nil {
					return nil, errors.WrapPathf(err, fmt.Sprintf("status.details[%d].'%s'", i, k), "invalid expect status detail")
				}

				statusDetailAssertions[i] = assert.AssertionFunc(func(v interface{}) error {
					m, ok := v.(proto.Message)
					if !ok {
						return fmt.Errorf("expect proto.Message but got %T", v)
					}
					if m == nil {
						return errors.New("got nil proto.Message")
					}
					if err := fullName.Assert(proto.MessageName(m)); err != nil {
						return err
					}
					if err := fields.Assert(v); err != nil {
						return errors.WithPath(err, fmt.Sprintf("'%s'", k))
					}
					return nil
				})
				break
			}
		}
	}
	return statusDetailAssertions, nil
}

func assertStatusCode(assertion assert.Assertion, sts *status.Status) error {
	err := assertion.Assert(sts.Code().String())
	if err == nil {
		return nil
	}
	return assertion.Assert(strconv.Itoa(int(sts.Code())))
}

func (e *Expect) assertStatusMessage(assertion assert.Assertion, sts *status.Status) error {
	if e.Status.Message == "" {
		return nil
	}
	err := assertion.Assert(sts.Message())
	if err == nil {
		return nil
	}
	return err
}

func (e *Expect) assertStatusDetails(assertions []assert.Assertion, sts *status.Status) error {
	if len(assertions) == 0 {
		return nil
	}

	actualDetails := sts.Details()

	for i, assertion := range assertions {
		if i >= len(actualDetails) {
			return errors.ErrorPath(fmt.Sprintf("details[%d]", i), `not found`)
		}

		if err := assertion.Assert(actualDetails[i]); err != nil {
			return errors.WithPath(err, fmt.Sprintf("details[%d]", i))
		}
	}

	return nil
}

func extract(v response) (proto.Message, *status.Status, error) {
	vs := v.rvalues
	if len(vs) != 2 {
		return nil, nil, errors.Errorf("expected return value length of method call is 2 but %d", len(vs))
	}

	if !vs[0].IsValid() {
		return nil, nil, errors.New("first return value is invalid")
	}
	message, ok := vs[0].Interface().(proto.Message)
	if !ok {
		if !vs[0].IsNil() {
			return nil, nil, errors.Errorf("expected first return value is proto.Message but %T", vs[0].Interface())
		}
	}

	if !vs[1].IsValid() {
		return nil, nil, errors.New("second return value is invalid")
	}
	callErr, ok := vs[1].Interface().(error)
	if !ok {
		if !vs[1].IsNil() {
			return nil, nil, errors.Errorf("expected second return value is error but %T", vs[1].Interface())
		}
	}
	var sts *status.Status
	if ok {
		sts, ok = status.FromError(callErr)
		if !ok {
			return nil, nil, errors.Errorf(`expected error is status but got %T: "%s"`, callErr, callErr.Error())
		}
	}

	return message, sts, nil
}
