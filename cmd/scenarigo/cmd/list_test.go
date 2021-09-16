package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/cobra"
)

func TestList(t *testing.T) {
	t.Run("use config", func(t *testing.T) {
		cmd := &cobra.Command{}
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		if err := listWithConfig(cmd, []string{}, "./testdata/scenarigo.yaml"); err != nil {
			t.Fatal(err)
		}
		expect := strings.TrimPrefix(`
testdata/scenarios/fail.yaml
testdata/scenarios/pass.yaml
`, "\n")
		if got := buf.String(); got != expect {
			dmp := diffmatchpatch.New()
			diffs := dmp.DiffMain(expect, got, false)
			t.Errorf("stdout differs:\n%s", dmp.DiffPrettyText(diffs))
		}
	})
	t.Run("specify by argument", func(t *testing.T) {
		cmd := &cobra.Command{}
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		if err := listWithConfig(cmd, []string{"testdata/scenarios/pass.yaml"}, ""); err != nil {
			t.Fatal(err)
		}
		expect := strings.TrimPrefix(`
testdata/scenarios/pass.yaml
`, "\n")
		if got := buf.String(); got != expect {
			dmp := diffmatchpatch.New()
			diffs := dmp.DiffMain(expect, got, false)
			t.Errorf("stdout differs:\n%s", dmp.DiffPrettyText(diffs))
		}
	})
	t.Run("override config by argument", func(t *testing.T) {
		cmd := &cobra.Command{}
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		if err := listWithConfig(cmd, []string{"testdata/scenarios/pass.yaml"}, "./testdata/scenarigo.yaml"); err != nil {
			t.Fatal(err)
		}
		expect := strings.TrimPrefix(`
testdata/scenarios/pass.yaml
`, "\n")
		if got := buf.String(); got != expect {
			dmp := diffmatchpatch.New()
			diffs := dmp.DiffMain(expect, got, false)
			t.Errorf("stdout differs:\n%s", dmp.DiffPrettyText(diffs))
		}
	})
}
