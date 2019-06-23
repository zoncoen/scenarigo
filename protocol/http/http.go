package http

import "github.com/zoncoen/scenarigo/protocol"

func init() {
	protocol.Register(&HTTP{})
}

// HTTP is a protocol type for the scenarigo step.
type HTTP struct{}

// Name implements protocol.Protocol interface.
func (p *HTTP) Name() string {
	return "http"
}

// UnmarshalRequest implements protocol.Protocol interface.
func (p *HTTP) UnmarshalRequest(unmarshal func(interface{}) error) (protocol.Invoker, error) {
	var r Request
	if err := unmarshal(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

// UnmarshalExpect implements protocol.Protocol interface.
func (p *HTTP) UnmarshalExpect(unmarshal func(interface{}) error) (protocol.AssertionBuilder, error) {
	var e Expect
	if err := unmarshal(&e); err != nil {
		return nil, err
	}
	return &e, nil
}
