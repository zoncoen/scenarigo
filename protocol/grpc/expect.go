package grpc

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/zoncoen/scenarigo/assert"
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/protocol"
	"github.com/zoncoen/yaml"
	"google.golang.org/grpc/status"
)

// Expect represents expected response values.
type Expect struct {
	Code string                          `yaml:"code"`
	Body yaml.KeyOrderPreservedInterface `yaml:"body"`
}

// Build implements protocol.AssertionBuilder interface.
func (e *Expect) Build(ctx *context.Context) (assert.Assertion, error) {
	expectBody, err := ctx.ExecuteTemplate(e.Body)
	if err != nil {
		return nil, errors.Errorf("invalid expect response: %s", err)
	}
	assertion := protocol.CreateAssertion(expectBody)

	return assert.AssertionFunc(func(v interface{}) error {
		message, callErr, err := extract(v)
		if err != nil {
			return err
		}
		if err := e.assertCode(callErr); err != nil {
			return err
		}
		if err := assertion.Assert(message); err != nil {
			return err
		}
		return nil
	}), nil
}

func (e *Expect) assertCode(err error) error {
	stErr, ok := status.FromError(err)
	if !ok {
		return errors.Errorf(`second return value is unknown type "%T"`, err)
	}

	expectedCode := "OK"
	if e.Code != "" {
		expectedCode = e.Code
	}
	if got, expected := stErr.Code().String(), expectedCode; got == expected {
		return nil
	}
	if got, expected := strconv.Itoa(int(int32(stErr.Code()))), expectedCode; got == expected {
		return nil
	}

	var details []string
	for _, i := range stErr.Details() {
		d, ok := i.(interface{ String() string })
		if ok {
			details = append(details, d.String())
			continue
		}
		e, ok := i.(interface{ Error() string })
		if ok {
			details = append(details, e.Error())
			continue
		}
	}

	return errors.Errorf(`expected code is "%s" but got "%s": %s: %s`, expectedCode, stErr.Code().String(), err, strings.Join(details, ", "))
}

func extract(v interface{}) (proto.Message, error, error) {
	vs, ok := v.([]reflect.Value)
	if !ok {
		return nil, nil, errors.Errorf("expected []reflect.Value but got %T", v)
	}
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

	return message, callErr, nil
}
