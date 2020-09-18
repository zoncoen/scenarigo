package grpc

import (
	"bytes"

	"github.com/goccy/go-yaml"
	"github.com/zoncoen/scenarigo/protocol"
)

func init() {
	protocol.Register(&GRPC{})
}

// GRPC is a protocol type for the scenarigo step.
type GRPC struct{}

// Name implements protocol.Protocol interface.
func (p *GRPC) Name() string {
	return "grpc"
}

// UnmarshalRequest implements protocol.Protocol interface.
func (p *GRPC) UnmarshalRequest(b []byte) (protocol.Invoker, error) {
	var r Request
	if err := yaml.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// UnmarshalExpect implements protocol.Protocol interface.
func (p *GRPC) UnmarshalExpect(b []byte) (protocol.AssertionBuilder, error) {
	var e Expect
	if b == nil {
		return &e, nil
	}
	decoder := yaml.NewDecoder(bytes.NewBuffer(b), yaml.UseOrderedMap(), yaml.Strict())
	if err := decoder.Decode(&e); err != nil {
		return nil, err
	}
	return &e, nil
}
