package main

import (
	"errors"

	"github.com/goccy/go-yaml"

	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/plugin"
	"github.com/zoncoen/scenarigo/schema"
)

var (
	DumpVarsStep                      = plugin.StepFunc(dumpVarsStep)
	Join         plugin.LeftArrowFunc = &join{}
)

func dumpVarsStep(ctx *context.Context, step *schema.Step) *context.Context {
	b, err := yaml.Marshal(step.Vars)
	if err != nil {
		ctx.Reporter().Errorf("failed to marshal vars: %s", err)
	}
	ctx.Reporter().Log(string(b))
	return ctx
}

type join struct{}

type joinArg struct {
	Prefix string `yaml:"prefix"`
	Text   string `yaml:"text"`
	Suffix string `yaml:"suffix"`
}

func (_ *join) Exec(in interface{}) (interface{}, error) {
	arg, ok := in.(*joinArg)
	if !ok {
		return nil, errors.New("arg must be a joinArg")
	}
	return arg.Prefix + arg.Text + arg.Suffix, nil
}

func (_ *join) UnmarshalArg(unmarshal func(interface{}) error) (interface{}, error) {
	var arg joinArg
	if err := unmarshal(&arg); err != nil {
		return nil, err
	}
	return &arg, nil
}
