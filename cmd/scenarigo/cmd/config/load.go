package config

import (
	"os"

	"github.com/fatih/color"
	"github.com/zoncoen/scenarigo/schema"
)

// ConfigPath will be set by the root command.
var ConfigPath string

// Load loads configuration from cfgpath.
func Load(cfgpath string) (*schema.Config, error) {
	if cfgpath != "" {
		return schema.LoadConfig(cfgpath, !color.NoColor)
	}
	cfg, err := schema.LoadConfig(DefaultConfigFileName, !color.NoColor)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil //nolint:nilnil
		}
		return nil, err
	}
	return cfg, nil
}
