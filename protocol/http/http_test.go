package http

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestHTTP_UnmarshalExpect(t *testing.T) {
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
				bytes: []byte(`code: 404`),
				expect: &Expect{
					Code: "404",
				},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				p := &HTTP{}
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
}
