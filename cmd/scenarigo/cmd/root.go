package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zoncoen/scenarigo/cmd/scenarigo/cmd/config"
)

const appName = "scenarigo"

func init() {
	rootCmd.PersistentFlags().StringVarP(&config.ConfigPath, "config", "c", "", "specify configuration file path")
}

var rootCmd = &cobra.Command{
	Use:   appName,
	Short: fmt.Sprintf("%s is a scenario-based API testing tool.", appName),
}

// Execute executes the root command.
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}
