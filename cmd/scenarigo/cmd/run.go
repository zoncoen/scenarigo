package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/zoncoen/scenarigo"
	"github.com/zoncoen/scenarigo/cmd/scenarigo/cmd/config"
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/reporter"
	"github.com/zoncoen/scenarigo/schema"
)

// ErrTestFailed is the error returned when the test failed.
var ErrTestFailed = errors.New("test failed")

var verbose bool

func init() {
	runCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "print verbose log")
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:           "run",
	Short:         "run test scenarios",
	Long:          "Runs test scenarios.",
	RunE:          run,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func run(cmd *cobra.Command, args []string) error {
	return runWithConfig(cmd, args, configFile)
}

func runWithConfig(cmd *cobra.Command, args []string, configPath string) error {
	opts := []func(*scenarigo.Runner) error{}
	cfg, err := loadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if cfg != nil {
		if len(args) > 0 {
			cfg.Scenarios = nil
		}
		opts = append(opts, scenarigo.WithConfig(cfg))
	}
	if len(args) > 0 {
		opts = append(opts, scenarigo.WithScenarios(args...))
	}
	r, err := scenarigo.NewRunner(opts...)
	if err != nil {
		return err
	}

	reporterOpts := []reporter.Option{
		reporter.WithWriter(cmd.OutOrStdout()),
	}
	if (cfg != nil && cfg.Output.Verbose) || verbose {
		reporterOpts = append(reporterOpts, reporter.WithVerboseLog())
	}

	noColor := color.NoColor
	if cfg != nil && cfg.Output.Colored != nil {
		noColor = !*cfg.Output.Colored
	}
	if noColor {
		reporterOpts = append(reporterOpts, reporter.WithNoColor())
	}

	success := reporter.Run(
		func(rptr reporter.Reporter) {
			r.Run(context.New(rptr))
		},
		reporterOpts...,
	)
	if !success {
		return ErrTestFailed
	}
	return nil
}

func loadConfig(cfgpath string) (*schema.Config, error) {
	if cfgpath != "" {
		return schema.LoadConfig(cfgpath, !color.NoColor)
	}
	cfg, err := schema.LoadConfig(config.DefaultConfigFileName, !color.NoColor)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return cfg, nil
}
