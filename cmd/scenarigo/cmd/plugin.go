package cmd

import (
	"github.com/spf13/cobra"
	sub "github.com/zoncoen/scenarigo/cmd/scenarigo/cmd/plugin"
)

var pluginCmd = &cobra.Command{
	Use:           "plugin",
	Short:         "provide operations for plugins",
	Long:          "Provides operations for plugins.",
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	for _, c := range sub.Commands() {
		pluginCmd.AddCommand(c)
	}
	rootCmd.AddCommand(pluginCmd)
}
