package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const appName = "scenarigo"

var configFile string

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "specify configuration file path")
}

var rootCmd = &cobra.Command{
	Use:   appName,
	Short: fmt.Sprintf("%s is a scenario-based API testing tool.", appName),
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}
