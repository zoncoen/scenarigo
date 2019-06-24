package grpc

import "github.com/zoncoen/scenarigo/protocol"

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
func (p *GRPC) UnmarshalRequest(unmarshal func(interface{}) error) (protocol.Invoker, error) {
	var r Request
	if err := unmarshal(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

// UnmarshalExpect implements protocol.Protocol interface.
func (p *GRPC) UnmarshalExpect(unmarshal func(interface{}) error) (protocol.AssertionBuilder, error) {
	var e Expect
	if err := unmarshal(&e); err != nil {
		return nil, err
	}
	return &e, nil
}
