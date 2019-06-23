package schema

import (
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/zoncoen/yaml"
)

// LoadScenarios loads test scenarios from path.
func LoadScenarios(path string) ([]*Scenario, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var scenarios []*Scenario
	d := yaml.NewDecoder(f)
	d.SetStrict(true)
	for {
		var s Scenario
		if err := d.Decode(&s); err != nil {
			if err == io.EOF {
				break
			}
			return nil, errors.Wrap(err, "failed to decode YAML")
		}
		s.filepath = path
		scenarios = append(scenarios, &s)
	}
	return scenarios, nil
}
