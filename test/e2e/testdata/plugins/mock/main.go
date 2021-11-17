package main

import (
	gocontext "context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/logger"
	"github.com/zoncoen/scenarigo/mock"
	"github.com/zoncoen/scenarigo/mock/protocol"
	"github.com/zoncoen/scenarigo/plugin"
)

func SetupServerFunc(filename string, ignoreMocksRemainError bool) plugin.SetupFunc {
	return func(ctx *plugin.Context) (*plugin.Context, func(*context.Context)) {
		teardown, err := runMockServer(filename, ignoreMocksRemainError)
		if err != nil {
			ctx.Reporter().Fatalf("failed to start mock server: %s", err)
		}
		return ctx, teardown
	}
}

func runMockServer(filename string, ignoreMocksRemainError bool) (func(*context.Context), error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var config mock.ServerConfig
	if err := yaml.NewDecoder(f, yaml.Strict()).Decode(&config); err != nil {
		return nil, err
	}
	srv, err := mock.NewServer(&config, logger.NewNopLogger())
	if err != nil {
		return nil, err
	}
	ch := make(chan error)
	go func() {
		ch <- srv.Start(gocontext.Background())
	}()
	ctx, cancel := gocontext.WithTimeout(gocontext.Background(), time.Second)
	defer cancel()
	if err := srv.Wait(ctx); err != nil {
		return nil, fmt.Errorf("failed to wait: %w", err)
	}
	addrs, err := srv.Addrs()
	if err != nil {
		return nil, err
	}
	for p, addr := range addrs {
		os.Setenv(fmt.Sprintf("TEST_%s_ADDR", strings.ToUpper(p)), addr)
	}
	return func(ctx *context.Context) {
		c, cancel := gocontext.WithTimeout(gocontext.Background(), time.Second)
		defer cancel()
		if err := srv.Stop(c); err != nil {
			mrerr := &protocol.MocksRemainError{}
			if errors.As(err, &mrerr) {
				if ignoreMocksRemainError {
					err = nil
				}
			}
			if err != nil {
				ctx.Reporter().Fatalf("failed to stop: %s", err)
			}
		}
		if err := <-ch; err != nil {
			ctx.Reporter().Fatalf("failed to start: %s", err)
		}
	}, nil
}
