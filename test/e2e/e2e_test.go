// +build !race

package scenarigo

import (
	"bytes"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/zoncoen/scenarigo"
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/internal/testutil"
	"github.com/zoncoen/scenarigo/reporter"
	"github.com/zoncoen/scenarigo/testdata/gen/pb/test"
	"google.golang.org/grpc"
)

func TestE2E(t *testing.T) {
	dir := "testdata/testcases"
	infos, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	files := []string{}
	for _, info := range infos {
		if info.IsDir() {
			continue
		}
		if strings.HasSuffix(info.Name(), ".yaml") {
			files = append(files, filepath.Join(dir, info.Name()))
		}
	}

	teardown := startGRPCServer(t)
	defer teardown()

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			f, err := os.Open(file)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			var tc TestCase
			if err := yaml.NewDecoder(f).Decode(&tc); err != nil {
				t.Fatal(err)
			}

			for _, scenario := range tc.Scenarios {
				t.Run(scenario.Filename, func(t *testing.T) {
					r, err := scenarigo.NewRunner(scenarigo.WithScenarios(filepath.Join(dir, "scenarios", scenario.Filename)))
					if err != nil {
						t.Fatal(err)
					}

					var b bytes.Buffer
					ok := reporter.Run(func(rptr reporter.Reporter) {
						r.Run(context.New(rptr).WithPluginDir("testdata/gen/plugins"))
					}, reporter.WithWriter(&b))
					if ok != scenario.Success {
						t.Errorf("expect %t but got %t", scenario.Success, ok)
					}

					f, err := os.Open(filepath.Join(dir, "stdout", scenario.Output.Stdout))
					if err != nil {
						t.Fatal(err)
					}
					defer f.Close()

					stdout, err := io.ReadAll(f)
					if err != nil {
						t.Fatal(err)
					}

					if expect, got := string(stdout), testutil.ResetDuration(b.String()); expect != got {
						t.Logf("===== expect =====\n%s", expect)
						t.Logf("===== got =====\n%s", got)
						t.FailNow()
					}
				})
			}
		})
	}
}

type TestCase struct {
	Tilte     string         `yaml:"title"`
	Scenarios []TestScenario `yaml:"scenarios"`
}

type TestScenario struct {
	Filename string       `yaml:"filename"`
	Success  bool         `yaml:"success"`
	Output   ExpectOutput `yaml:"output"`
}

type ExpectOutput struct {
	Stdout string `yaml:"stdout"`
}

func startGRPCServer(t *testing.T) func() {
	t.Helper()

	token := "XXXXX"
	testServer := &testGRPCServer{
		users: map[string]string{
			token: "test user",
		},
	}
	s := grpc.NewServer()
	test.RegisterTestServer(s, testServer)

	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if err := os.Setenv("TEST_GRPC_SERVER_ADDR", ln.Addr().String()); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := os.Setenv("TEST_TOKEN", token); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	go func() {
		_ = s.Serve(ln)
	}()

	return func() {
		s.Stop()
		os.Unsetenv("TEST_GRPC_SERVER_ADDR")
		os.Unsetenv("TEST_TOKEN")
	}
}
