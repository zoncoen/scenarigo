package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/zoncoen/scenarigo/cmd/scenarigo/cmd"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := cmd.Execute(ctx); err != nil {
		if err == cmd.ErrTestFailed {
			os.Exit(10)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
