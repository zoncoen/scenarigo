package scenarigo

import (
	"sort"

	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/plugin"
)

type setupMap map[string]plugin.SetupFunc

func (sm setupMap) setup(ctx *plugin.Context) (*plugin.Context, func(*plugin.Context)) {
	if len(sm) == 0 {
		return ctx, nil
	}
	var keys sort.StringSlice
	teardowns := map[string]func(*plugin.Context){}
	setupCtx := ctx
	ctx.Run("setup", func(ctx *plugin.Context) {
		for key := range sm {
			keys = append(keys, key)
		}
		sort.Sort(keys)
		for _, key := range keys {
			if ctx.Reporter().Failed() {
				break
			}
			newCtx := ctx
			ctx.Run(key, func(ctx *context.Context) {
				ctx, teardown := sm[key](ctx)
				if ctx != nil {
					newCtx = ctx
				}
				if teardown != nil {
					teardowns[key] = teardown
				}
			})
			ctx = newCtx.WithReporter(ctx.Reporter())
		}
		setupCtx = ctx
	})
	ctx = setupCtx.WithReporter(ctx.Reporter())
	if len(teardowns) == 0 {
		return ctx, nil
	}
	return ctx, func(ctx *plugin.Context) {
		ctx.Run("teardown", func(ctx *plugin.Context) {
			for i := keys.Len() - 1; i >= 0; i-- {
				key := keys[i]
				if teardown, ok := teardowns[key]; ok {
					ctx.Run(key, func(ctx *context.Context) {
						teardown(ctx)
					})
				}
			}
		})
	}
}
