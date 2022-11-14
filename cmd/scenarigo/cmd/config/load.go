package config

import (
	"os"

	"github.com/zoncoen/scenarigo/schema"
)

// ConfigPath will be set by the root command.
var ConfigPath string

// Load loads configuration from cfgpath.
func Load(cfgpath string) (*schema.Config, error) {
	if cfgpath != "" {
		return schema.LoadConfig(cfgpath)
	}
	cfg, err := schema.LoadConfig(DefaultConfigFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil //nolint:nilnil
		}
		return nil, err
	}
	return cfg, nil
}
