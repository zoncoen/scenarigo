package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/zoncoen/scenarigo"
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/reporter"
)

var (
	// ErrTestFailed is the error returned when the test failed.
	ErrTestFailed = errors.New("test failed")
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:           "run",
	Short:         "run test scenarios",
	Args:          cobra.MinimumNArgs(1),
	RunE:          run,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func run(cmd *cobra.Command, args []string) error {
	opts := []func(*scenarigo.Runner) error{}
	for _, arg := range args {
		opts = append(opts, scenarigo.WithScenarios(arg))
	}
	r, err := scenarigo.NewRunner(opts...)
	if err != nil {
		return err
	}
	success := reporter.Run(
		func(rptr reporter.Reporter) {
			r.Run(context.New(rptr))
		},
		reporter.WithWriter(os.Stdout),
	)
	if !success {
		return ErrTestFailed
	}
	return nil
}
