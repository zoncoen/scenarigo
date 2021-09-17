package cmd

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/cobra"
	"github.com/zoncoen/scenarigo/cmd/scenarigo/cmd/config"
	"github.com/zoncoen/scenarigo/internal/testutil"
)

func TestRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		w.Header().Add("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	os.Setenv("TEST_ADDR", srv.URL)
	defer os.Unsetenv("TEST_ADDR")

	tests := map[string]struct {
		args         []string
		config       string
		expectError  bool
		expectOutput string
	}{
		"specify by argument": {
			args:        []string{"testdata/scenarios/pass.yaml"},
			expectError: false,
			expectOutput: strings.TrimPrefix(`
ok  	testdata/scenarios/pass.yaml	0.000s
`, "\n"),
		},
		"use config": {
			args:        []string{},
			config:      "./testdata/scenarigo.yaml",
			expectError: true,
			expectOutput: strings.TrimPrefix(`
--- FAIL: scenarios/fail.yaml (0.00s)
    --- FAIL: scenarios/fail.yaml//echo (0.00s)
        --- FAIL: scenarios/fail.yaml//echo/POST_/echo (0.00s)
                [0] send request
                request:
                  method: POST
                  url: http://127.0.0.1:12345/echo
                  header:
                    User-Agent:
                    - scenarigo/v1.0.0
                  body:
                    message: request
                response:
                  header:
                    Content-Length:
                    - "23"
                    Content-Type:
                    - application/json
                    Date:
                    - Mon, 01 Jan 0001 00:00:00 GMT
                  body:
                    message: request
                elapsed time: 0.000000 sec
                  12 |   expect:
                  13 |     code: 200
                  14 |     body:
                > 15 |       message: "response"
                                      ^
                expected response but got request
FAIL
FAIL	scenarios/fail.yaml	0.000s
FAIL
ok  	scenarios/pass.yaml	0.000s
`, "\n"),
		},
		"override config by argument": {
			config:      "./testdata/scenarigo.yaml",
			args:        []string{"testdata/scenarios/pass.yaml"},
			expectError: false,
			expectOutput: strings.TrimPrefix(`
ok  	scenarios/pass.yaml	0.000s
`, "\n"),
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			cmd := &cobra.Command{}
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			config.ConfigPath = test.config
			err := run(cmd, test.args)
			if test.expectError && err == nil {
				t.Fatal("expect error but no error")
			}
			if !test.expectError && err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got, expect := testutil.ReplaceOutput(buf.String()), test.expectOutput; got != expect {
				dmp := diffmatchpatch.New()
				diffs := dmp.DiffMain(expect, got, false)
				t.Errorf("stdout differs:\n%s", dmp.DiffPrettyText(diffs))
			}
		})
	}
}
