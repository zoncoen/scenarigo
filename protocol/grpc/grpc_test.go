package grpc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGRPC_UnmarshalRequest(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		tests := map[string]struct {
			bytes  []byte
			expect *Request
		}{
			"default": {
				bytes:  nil,
				expect: &Request{},
			},
			"method": {
				bytes: []byte(`method: Ping`),
				expect: &Request{
					Method: "Ping",
				},
			},
			"message": {
				bytes: []byte(`message: hello`),
				expect: &Request{
					Message: "hello",
				},
			},
			"body (check backward compatibility)": {
				bytes: []byte(`body: hello`),
				expect: &Request{
					Message: "hello",
				},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				p := &GRPC{}
				invoker, err := p.UnmarshalRequest(test.bytes)
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if diff := cmp.Diff(test.expect, invoker); diff != "" {
					t.Errorf("request differs (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("ng", func(t *testing.T) {
		tests := map[string]struct {
			bytes []byte
		}{
			"use body and message": {
				bytes: []byte(`
body: test
message: test`),
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				p := &GRPC{}
				_, err := p.UnmarshalRequest(test.bytes)
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
			})
		}
	})
}

func TestGRPC_UnmarshalExpect(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		tests := map[string]struct {
			bytes  []byte
			expect *Expect
		}{
			"default": {
				bytes:  nil,
				expect: &Expect{},
			},
			"code": {
				bytes: []byte(`code: InvalidArgument`),
				expect: &Expect{
					Code: "InvalidArgument",
				},
			},
			"message": {
				bytes: []byte(`message: hello`),
				expect: &Expect{
					Message: "hello",
				},
			},
			"body (check backward compatibility)": {
				bytes: []byte(`body: hello`),
				expect: &Expect{
					Message: "hello",
				},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				p := &GRPC{}
				builder, err := p.UnmarshalExpect(test.bytes)
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if diff := cmp.Diff(test.expect, builder); diff != "" {
					t.Errorf("request differs (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("ng", func(t *testing.T) {
		tests := map[string]struct {
			bytes []byte
		}{
			"unknown field": {
				bytes: []byte(`a: b`),
			},
			"duplicated field": {
				bytes: []byte("code: InvalidArgument\ncode: InvalidArgument"),
			},
			"use body and message": {
				bytes: []byte(`
body: test
message: test`),
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				p := &GRPC{}
				_, err := p.UnmarshalExpect(test.bytes)
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
			})
		}
	})
}
