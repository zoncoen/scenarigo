// Package schema provides the test scenario data schema for scenarigo.
package schema

// Scenario represents a test scenario.
type Scenario struct {
	Title       string                 `yaml:"title"`
	Description string                 `yaml:"description"`
	Vars        map[string]interface{} `yaml:"vars"`
	Steps       []*Step                `yaml:"steps"`

	filepath string // YAML filepath
}

// Filepath returns YAML filepath of s.
func (s *Scenario) Filepath() string {
	return s.filepath
}

// Step represents a step of scenario.
type Step struct {
	Title       string                 `yaml:"title"`
	Description string                 `yaml:"description"`
	Vars        map[string]interface{} `yaml:"vars"`
	Protocol    string                 `yaml:"protocol"`
	Request     interface{}            `yaml:"request"`
	Expect      interface{}            `yaml:"expect"`
	Bind        Bind                   `yaml:"bind"`
}

// Bind represents bindings of variables.
type Bind struct {
	Vars map[string]interface{} `yaml:"vars"`
}
