package context_test

import (
	"testing"
	"time"

	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/token"
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/schema"
)

func TestContext(t *testing.T) {
	t.Run("ScenarioFilepath", func(t *testing.T) {
		path := "test.yaml"
		ctx := context.FromT(t).WithScenarioFilepath(path)
		if got := ctx.ScenarioFilepath(); got != path {
			t.Errorf("expect %q but got %q", path, got)
		}
	})
	t.Run("node", func(t *testing.T) {
		ctx := context.FromT(t)
		node := ast.String(token.String("", "", nil))
		ctx = ctx.WithNode(node)
		if ctx.Node() != node {
			t.Fatal("failed to get node")
		}
	})
	t.Run("enabledColor", func(t *testing.T) {
		ctx := context.FromT(t)
		ctx = ctx.WithEnabledColor(true)
		if !ctx.EnabledColor() {
			t.Fatal("failed to get enabledColor")
		}
	})
}

func TestRunWithRetry(t *testing.T) {
	ctx := context.FromT(t)
	interval := schema.Duration(time.Millisecond)
	maxRetries := 1
	var i int
	context.RunWithRetry(ctx, "sub", func(ctx *context.Context) {
		i++
		if i > 1 {
			return
		}
		ctx.Reporter().FailNow()
	}, &schema.RetryPolicyConstant{
		Interval:   &interval,
		MaxRetries: &maxRetries,
	})
}
