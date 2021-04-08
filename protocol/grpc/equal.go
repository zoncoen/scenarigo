package grpc

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/zoncoen/scenarigo/assert"
)

func init() {
	assert.RegisterCustomEqualer(assert.EqualerFunc(equalEnum))
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
