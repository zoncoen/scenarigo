package schema

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/parser"
	"github.com/zoncoen/scenarigo/errors"
)

var scehamaVersionPath *yaml.Path

func init() {
	p, err := yaml.PathString("$.schemaVersion")
	if err != nil {
		panic(fmt.Sprintf("YAML parser error: %s", err))
	}
	scehamaVersionPath = p
}

// Config represents a configuration.
type Config struct {
	SchemaVersion   string       `yaml:"schemaVersion,omitempty"`
	Scenarios       []string     `yaml:"scenarios,omitempty"`
	PluginDirectory string       `yaml:"pluginDirectory,omitempty"`
	Output          OutputConfig `yaml:"output,omitempty"`

	Root string `yaml:"-"`
}

// OutputConfig represents a output configuration.
type OutputConfig struct {
	Verbose bool         `yaml:"verbose,omitempty"`
	Colored *bool        `yaml:"colored,omitempty"`
	Report  ReportConfig `yaml:"report,omitempty"`
}

// ReportConfig represents a report configuration.
type ReportConfig struct {
	JSON  JSONReportConfig  `yaml:"json,omitempty"`
	JUnit JUnitReportConfig `yaml:"junit,omitempty"`
}

// JSONReportConfig represents a JSON report configuration.
type JSONReportConfig struct {
	Filename string `yaml:"filename,omitempty"`
}

// JUnitReportConfig represents a JUnit report configuration.
type JUnitReportConfig struct {
	Filename string `yaml:"filename,omitempty"`
}

// LoadConfig loads a configuration from path.
func LoadConfig(path string, colored bool) (*Config, error) {
	r, err := os.OpenFile(path, os.O_RDONLY, 0o400)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	f, err := parser.ParseBytes(b, 0)
	if err != nil {
		return nil, err
	}
	if len(f.Docs) == 0 {
		return nil, errors.New("schemaVersion not found")
	}

	vnode, err := scehamaVersionPath.FilterNode(f.Docs[0].Body)
	if err != nil {
		return nil, err
	}
	if vnode == nil {
		return nil, errors.New("schemaVersion not found")
	}

	var v string
	if err := yaml.NodeToValue(vnode, &v); err != nil {
		return nil, errors.WithNodeAndColored(
			errors.ErrorPathf("schemaVersion", "invalid version: %s", err),
			f.Docs[0].Body,
			colored,
		)
	}

	root, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, fmt.Errorf("failed to get root directory: %w", err)
	}

	switch v {
	case "config/v1":
		var cfg Config
		if err := yaml.NodeToValue(f.Docs[0].Body, &cfg, yaml.Strict()); err != nil {
			return nil, err
		}
		cfg.Root = root
		return &cfg, nil
	default:
		return nil, errors.WithNodeAndColored(
			errors.ErrorPathf("schemaVersion", "unknown version %q", v),
			f.Docs[0].Body,
			colored,
		)
	}
}
