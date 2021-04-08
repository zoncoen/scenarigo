package grpc

import (
	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/zoncoen/scenarigo/assert"
)

func init() {
	assert.RegisterCustomEqualer(assert.EqualerFunc(equalEnum))
	assert.RegisterCustomEqualer(assert.EqualerFunc(equalMessage))
}

func equalEnum(expected interface{}, got interface{}) (bool, error) {
	s, ok := expected.(string)
	if !ok {
		return false, nil
	}
	enum, ok := got.(protoreflect.Enum)
	if !ok {
		return false, nil
	}
	if string(enum.Descriptor().Values().ByNumber(enum.Number()).Name()) == s {
		return true, nil
	}
	return false, nil
}

func equalMessage(expected interface{}, got interface{}) (bool, error) {
	em, ok := expected.(proto.Message)
	if !ok {
		return false, nil
	}
	gm, ok := got.(proto.Message)
	if !ok {
		return false, nil
	}
	if proto.Equal(em, gm) {
		return true, nil
	}
	return false, nil
}
