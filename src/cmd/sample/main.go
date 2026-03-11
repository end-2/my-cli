package main

import (
	"fmt"
	"io"
	"os"

	"github.com/end-2/my-cli/src/pkg/cliutil"
	"github.com/spf13/cobra"
)

var Version = "dev"

type Dependencies struct {
	LoadConfig func() (Config, error)
}

type executor struct {
	version string
	deps    Dependencies
}

func main() {
	if err := execute(os.Stdout, os.Stderr, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func execute(stdout, stderr io.Writer, args []string) error {
	return executeWithDependencies(stdout, stderr, args, Dependencies{})
}

func executeWithDependencies(stdout, stderr io.Writer, args []string, deps Dependencies) error {
	if deps.LoadConfig == nil {
		deps.LoadConfig = loadConfig
	}

	cmd := newExecutor(Version, deps).newRootCmd(stdout, stderr)
	cmd.SetArgs(cliutil.NormalizeLongFlags(args, "version", "help", "dry-run"))
	return cmd.Execute()
}

func newExecutor(version string, deps Dependencies) *executor {
	return &executor{
		version: version,
		deps:    deps,
	}
}

func (e *executor) newRootCmd(stdout, stderr io.Writer) *cobra.Command {
	var showVersion bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:           "sample",
		Short:         "Sample CLI application",
		Long:          "sample is a small Cobra-based CLI example for my-cli.\nOverride message and dry_run_message with sample.yaml via src/pkg/config.",
		Example:       "sample\nsample --dry-run\nsample --version\nsample --help",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), e.version)
				return err
			}

			config, err := e.deps.LoadConfig()
			if err != nil {
				return err
			}

			if dryRun {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), config.DryRunMessage)
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), config.Message)
			return err
		},
	}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.Flags().BoolVarP(&showVersion, "version", "v", false, "print binary version")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "preview the command without running it")

	return cmd
}
