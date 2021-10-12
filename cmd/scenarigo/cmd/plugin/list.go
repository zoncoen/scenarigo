package plugin

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"github.com/zoncoen/scenarigo/cmd/scenarigo/cmd/config"
	"github.com/zoncoen/scenarigo/internal/filepathutil"
)

var listCmd = &cobra.Command{
	Use:           "list",
	Short:         "list the plugins",
	Long:          "Lists the plugins as relative paths from the current directory.",
	Args:          cobra.ExactArgs(0),
	RunE:          list,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func list(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(config.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if cfg == nil {
		return errors.New("config file not found")
	}
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	pluginDir := filepathutil.From(cfg.Root, cfg.PluginDirectory)
	var plugins sort.StringSlice
	for out := range cfg.Plugins {
		rel, err := filepath.Rel(wd, filepathutil.From(pluginDir, out))
		if err != nil {
			return fmt.Errorf("failed to get releative path: %w", err)
		}
		plugins = append(plugins, rel)
	}
	sort.Sort(plugins)
	for _, p := range plugins {
		fmt.Fprintln(cmd.OutOrStdout(), p)
	}
	return nil
}
