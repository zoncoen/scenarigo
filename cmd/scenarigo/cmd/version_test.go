package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/cobra"
	"github.com/zoncoen/scenarigo/version"
)

func TestVersion(t *testing.T) {
	var b bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&b)
	printVersion(cmd, nil)
	if got, expect := b.String(), fmt.Sprintf("%s version %s\n", appName, version.String()); got != expect {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(expect, got, false)
		t.Errorf("output differs:\n%s", dmp.DiffPrettyText(diffs))
	}
}
