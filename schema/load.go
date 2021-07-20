package schema

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/pkg/errors"
)

// LoadScenarios loads test scenarios from path.
func LoadScenarios(path string) ([]*Scenario, error) {
	f, err := parser.ParseFile(path, 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse YAML")
	}
	return loadScenarios(f)
}

// LoadScenariosFromReader loads test scenarios with io.Reader.
func LoadScenariosFromReader(r io.Reader) ([]*Scenario, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read")
	}
	f, err := parser.ParseBytes(b, 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse YAML")
	}
	return loadScenarios(f)
}

func loadScenarios(f *ast.File) ([]*Scenario, error) {
	var buf bytes.Buffer
	dec := yaml.NewDecoder(&buf, yaml.UseOrderedMap(), yaml.Strict())
	var scenarios []*Scenario
	for _, doc := range f.Docs {
		var s Scenario
		if err := dec.DecodeFromNode(doc.Body, &s); err != nil {
			return nil, errors.Wrap(err, "failed to decode YAML")
		}
		s.filepath = f.Name
		s.Node = doc.Body
		scenarios = append(scenarios, &s)
	}
	return scenarios, nil
}
