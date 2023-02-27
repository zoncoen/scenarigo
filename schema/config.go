package schema

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"golang.org/x/mod/module"

	"github.com/zoncoen/scenarigo/errors"
	"github.com/zoncoen/scenarigo/internal/filepathutil"
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
	SchemaVersion   string          `yaml:"schemaVersion,omitempty"`
	Scenarios       []string        `yaml:"scenarios,omitempty"`
	PluginDirectory string          `yaml:"pluginDirectory,omitempty"`
	Plugins         PluginConfigMap `yaml:"plugins,omitempty"`
	Output          OutputConfig    `yaml:"output,omitempty"`

	// absolute path to the configuration file
	Root     string          `yaml:"-"`
	Comments yaml.CommentMap `yaml:"-"`
}

// PluginConfigMap represents a plugin configurations.
type PluginConfigMap map[string]PluginConfig

func (m *PluginConfigMap) UnmarshalYAML(b []byte) error {
	var s yaml.MapSlice
	if err := yaml.Unmarshal(b, &s); err != nil {
		return err
	}
	configs := map[string]PluginConfig{}
	for i, item := range s {
		name, err := toYAMLString(item.Key)
		if err != nil {
			return err
		}
		val, ok := item.Value.(map[string]interface{})
		if !ok {
			return errors.Errorf("%T was used where mapping is expected", item.Value)
		}
		var src string
		for k, v := range val {
			switch k {
			case "src":
				src, err = toYAMLString(v)
				if err != nil {
					return err
				}
			default:
				return errors.Errorf(`unknown field "%s"`, k)
			}
		}
		configs[name] = PluginConfig{
			Order: i + 1,
			Name:  name,
			Src:   src,
		}
	}
	*m = configs
	return nil
}

func toYAMLString(in interface{}) (string, error) {
	switch v := in.(type) {
	case string:
		return v, nil
	case uint64:
		return strconv.Itoa(int(v)), nil
	case int64:
		return strconv.Itoa(int(v)), nil
	}
	return "", errors.Errorf("value of type %T is not assignable to type string", in)
}

func (m PluginConfigMap) MarshalYAML() ([]byte, error) {
	var s yaml.MapSlice
	for _, c := range m.ToSlice() {
		s = append(s, yaml.MapItem{
			Key:   c.Name,
			Value: c,
		})
	}
	return yaml.Marshal(s)
}

// ToSlice returns m as a slice.
func (m PluginConfigMap) ToSlice() []PluginConfig {
	s := []PluginConfig{}
	for _, p := range m {
		s = append(s, p)
	}
	sort.Slice(s, func(i, j int) bool {
		return s[i].Order < s[j].Order
	})
	return s
}

// PluginConfig represents a plugin configuration.
type PluginConfig struct {
	Order int    `yaml:"-"`
	Name  string `yaml:"-"`
	Src   string `yaml:"src,omitempty"`
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
func LoadConfig(path string) (*Config, error) {
	r, err := os.OpenFile(path, os.O_RDONLY, 0o400)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	root, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, fmt.Errorf("failed to get root directory: %w", err)
	}

	return LoadConfigFromReader(r, root)
}

// LoadConfigFromReader loads a configuration from r.
func LoadConfigFromReader(r io.Reader, root string) (*Config, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	f, err := parser.ParseBytes(b, 1)
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
		return nil, fmt.Errorf("invalid version: %w", err)
	}

	switch v {
	case "config/v1":
		var cfg Config
		cm := make(yaml.CommentMap)
		if err := yaml.NodeToValue(f.Docs[0].Body, &cfg, yaml.Strict(), yaml.CommentToMap(cm)); err != nil {
			return nil, err
		}
		cfg.Root = root
		if len(cm) > 0 {
			cfg.Comments = cm
		}
		if err := validate(&cfg, f.Docs[0].Body); err != nil {
			return nil, err
		}
		return &cfg, nil
	default:
		return nil, errors.WithNodeAndColored(
			errors.ErrorPathf("schemaVersion", "unknown version %q", v),
			f.Docs[0].Body,
			!color.NoColor,
		)
	}
}

func validate(c *Config, node ast.Node) error {
	var errs []error
	for i, p := range c.Scenarios {
		if err := stat(c, p, (&yaml.PathBuilder{}).Root().Child("scenarios").Index(uint(i)).Build(), node); err != nil {
			errs = append(errs, err)
		}
	}
	for name, p := range c.Plugins {
		p := p
		if err := stat(c, p.Src, (&yaml.PathBuilder{}).Root().Child("plugins").Child(name).Child("src").Build(), node); err != nil {
			var neErr notExist
			if errors.As(err, &neErr) {
				m := p.Src
				if i := strings.Index(m, "@"); i >= 0 { // trim version query
					m = p.Src[:i]
				}
				if merr := module.CheckPath(m); merr != nil {
					err = fmt.Errorf("%s: %w", err, merr)
				} else {
					err = nil
				}
			}
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Errors(errs...)
}

type notExist error

func stat(c *Config, p string, path *yaml.Path, node ast.Node) error {
	if _, err := os.Stat(filepathutil.From(c.Root, p)); err != nil {
		if os.IsNotExist(err) {
			err = notExist(errors.Errorf("%s: no such file or directory", p))
		}
		return errors.WithNodeAndColored(
			errors.WithPath(err, strings.TrimPrefix(path.String(), "$.")),
			node, !color.NoColor,
		)
	}
	return nil
}
