// Package schema provides the test scenario data schema for scenarigo.
package schema

import (
	"github.com/goccy/go-yaml/ast"
	"github.com/pkg/errors"

	"github.com/zoncoen/scenarigo/protocol"
)

// Scenario represents a test scenario.
type Scenario struct {
	Title       string                 `yaml:"title,omitempty"`
	Description string                 `yaml:"description,omitempty"`
	Plugins     map[string]string      `yaml:"plugins,omitempty"`
	Vars        map[string]interface{} `yaml:"vars,omitempty"`
	Steps       []*Step                `yaml:"steps,omitempty"`

	// The strict YAML decoder fails to decode if finds an unknown field.
	// Anchors is the field for enabling to define YAML anchors by avoiding the error.
	// This field doesn't need to hold some data because anchors expand by the decoder.
	Anchors anchors `yaml:"anchors,omitempty"`

	filepath string   // YAML filepath
	Node     ast.Node `yaml:"-"`
}

// Filepath returns YAML filepath of s.
func (s *Scenario) Filepath() string {
	return s.filepath
}

// Step represents a step of scenario.
type Step struct {
	Title       string                    `yaml:"title,omitempty"`
	Description string                    `yaml:"description,omitempty"`
	Vars        map[string]interface{}    `yaml:"vars,omitempty"`
	Protocol    string                    `yaml:"protocol,omitempty"`
	Request     protocol.Invoker          `yaml:"request,omitempty"`
	Expect      protocol.AssertionBuilder `yaml:"expect,omitempty"`
	Include     string                    `yaml:"include,omitempty"`
	Ref         interface{}               `yaml:"ref,omitempty"`
	Bind        Bind                      `yaml:"bind,omitempty"`
	Retry       *RetryPolicy              `yaml:"retry,omitempty"`
}

type rawMessage []byte

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (r *rawMessage) UnmarshalYAML(b []byte) error {
	*r = b
	return nil
}

type stepUnmarshaller struct {
	Title       string                 `yaml:"title,omitempty"`
	Description string                 `yaml:"description,omitempty"`
	Vars        map[string]interface{} `yaml:"vars,omitempty"`
	Protocol    string                 `yaml:"protocol,omitempty"`
	Include     string                 `yaml:"include,omitempty"`
	Ref         interface{}            `yaml:"ref,omitempty"`
	Bind        Bind                   `yaml:"bind,omitempty"`
	Retry       *RetryPolicy           `yaml:"retry,omitempty"`

	Request rawMessage `yaml:"request,omitempty"`
	Expect  rawMessage `yaml:"expect,omitempty"`
}

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (s *Step) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// unmarshal into stepUnmarshaller instead of Step for dynamic unmarshalling Request/Expect
	var unmarshaled stepUnmarshaller
	if err := unmarshal(&unmarshaled); err != nil {
		return err
	}

	s.Title = unmarshaled.Title
	s.Description = unmarshaled.Description
	s.Vars = unmarshaled.Vars
	s.Protocol = unmarshaled.Protocol
	s.Include = unmarshaled.Include
	s.Ref = unmarshaled.Ref
	s.Bind = unmarshaled.Bind
	s.Retry = unmarshaled.Retry

	p := protocol.Get(s.Protocol)
	if p == nil {
		if unmarshaled.Request != nil || unmarshaled.Expect != nil {
			return errors.Errorf("unknown protocol: %s", s.Protocol)
		}
		return nil
	}
	if unmarshaled.Request != nil {
		invoker, err := p.UnmarshalRequest(unmarshaled.Request)
		if err != nil {
			return err
		}
		s.Request = invoker
	}
	builder, err := p.UnmarshalExpect(unmarshaled.Expect)
	if err != nil {
		return err
	}
	s.Expect = builder

	return nil
}

// Bind represents bindings of variables.
type Bind struct {
	Vars map[string]interface{} `yaml:"vars"`
}

type anchors struct{}

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (a anchors) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return nil
}
