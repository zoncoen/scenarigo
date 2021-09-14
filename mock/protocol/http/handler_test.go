package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"

	"github.com/zoncoen/scenarigo/logger"
	"github.com/zoncoen/scenarigo/mock/protocol"
)

func TestHandler(t *testing.T) {
	type expect struct {
		code   int
		header http.Header
		body   string
	}
	type step struct {
		request *http.Request
		expect  *expect
	}
	t.Run("success", func(t *testing.T) {
		tests := map[string]struct {
			filename string
			steps    []step
		}{
			"minimal": {
				filename: "testdata/http.yaml",
				steps: []step{
					{
						request: httptest.NewRequest("GET", "/", nil),
						expect: &expect{
							code:   200,
							header: http.Header{},
							body:   `{"message": "hello"}`,
						},
					},
				},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				f, err := os.Open(test.filename)
				if err != nil {
					t.Fatal(err)
				}
				defer f.Close()
				var mocks []protocol.Mock
				if err := yaml.NewDecoder(f).Decode(&mocks); err != nil {
					t.Fatal(err)
				}
				iter := protocol.NewMockIterator(mocks)
				h := NewHandler(iter, logger.NewNopLogger())
				for _, step := range test.steps {
					rec := httptest.NewRecorder()
					h.ServeHTTP(rec, step.request)
					if got, expect := rec.Code, step.expect.code; got != expect {
						t.Errorf("expect code %d but got %d", expect, got)
					}
					if diff := cmp.Diff(step.expect.header, rec.Header()); diff != "" {
						t.Errorf("body differs (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(step.expect.body, strings.TrimSuffix(rec.Body.String(), "\n")); diff != "" {
						t.Errorf("body differs (-want +got):\n%s", diff)
					}
				}
				if err := iter.Stop(); err != nil {
					t.Fatalf("failed to stop iterator: %s", err)
				}
			})
		}
	})
	t.Run("faulure", func(t *testing.T) {
		tests := map[string]struct {
			filename string
			steps    []step
		}{
			"invalid protocol": {
				filename: "testdata/invalid-protocol.yaml",
				steps: []step{
					{
						request: httptest.NewRequest("GET", "/", nil),
						expect: &expect{
							code: 500,
							header: http.Header{
								"Content-Type": []string{"text/plain; charset=utf-8"},
							},
							body: `received HTTP request but the mock protocol is "invalid"`,
						},
					},
				},
			},
			"over request": {
				filename: "testdata/http.yaml",
				steps: []step{
					{
						request: httptest.NewRequest("GET", "/", nil),
						expect: &expect{
							code:   200,
							header: http.Header{},
							body:   `{"message": "hello"}`,
						},
					},
					{
						request: httptest.NewRequest("GET", "/", nil),
						expect: &expect{
							code: 500,
							header: http.Header{
								"Content-Type": []string{"text/plain; charset=utf-8"},
							},
							body: "no mocks remain",
						},
					},
				},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				f, err := os.Open(test.filename)
				if err != nil {
					t.Fatal(err)
				}
				defer f.Close()
				var mocks []protocol.Mock
				if err := yaml.NewDecoder(f).Decode(&mocks); err != nil {
					t.Fatal(err)
				}
				iter := protocol.NewMockIterator(mocks)
				h := NewHandler(iter, logger.NewNopLogger())
				for _, step := range test.steps {
					rec := httptest.NewRecorder()
					h.ServeHTTP(rec, step.request)
					if got, expect := rec.Code, step.expect.code; got != expect {
						t.Errorf("expect code %d but got %d", expect, got)
					}
					if diff := cmp.Diff(step.expect.header, rec.Header()); diff != "" {
						t.Errorf("body differs (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(step.expect.body, strings.TrimSuffix(rec.Body.String(), "\n")); diff != "" {
						t.Errorf("body differs (-want +got):\n%s", diff)
					}
				}
				if err := iter.Stop(); err != nil {
					t.Fatalf("failed to stop iterator: %s", err)
				}
			})
		}
	})
}

func TestExtract(t *testing.T) {
	t.Run("failure", func(t *testing.T) {
		tests := map[string]struct {
			resp *HTTPResponse
		}{
			"invalid status code": {
				resp: &HTTPResponse{
					Code: "OK",
				},
			},
			"invalid header key": {
				resp: &HTTPResponse{
					Header: yaml.MapSlice{
						yaml.MapItem{
							Key: nil,
						},
					},
				},
			},
			"failed to marshal response body": {
				resp: &HTTPResponse{
					Header: yaml.MapSlice{
						yaml.MapItem{
							Key:   "Content-Type",
							Value: "text/plain",
						},
					},
					Body: t,
				},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				if _, _, _, err := test.resp.extract(); err == nil {
					t.Fatalf("no error")
				}
			})
		}
	})
}
