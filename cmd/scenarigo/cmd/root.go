package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const appName = "scenarigo"

var rootCmd = &cobra.Command{
	Use:   appName,
	Short: fmt.Sprintf("%s is a scenario testing tool for APIs.", appName),
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}
