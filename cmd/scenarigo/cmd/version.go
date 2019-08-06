package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version string = "0.1.0"

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: fmt.Sprintf("print %s version", appName),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s version %s\n", appName, version)
	},
}
