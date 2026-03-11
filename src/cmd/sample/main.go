package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var Version = "dev"

func main() {
	if err := execute(os.Stdout, os.Stderr, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func execute(stdout, stderr io.Writer, args []string) error {
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs(normalizeArgs(args))

	return cmd.Execute()
}

func newRootCmd(stdout, stderr io.Writer) *cobra.Command {
	var showVersion bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:           "sample",
		Short:         "Sample CLI application",
		Long:          "sample is a small Cobra-based CLI example for my-cli.",
		Example:       "sample\nsample --dry-run\nsample --version\nsample --help",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), Version)
				return err
			}

			if dryRun {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "Dry run: would print Hello MY CLI")
				return err
			}

			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Hello MY CLI")
			return err
		},
	}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.Flags().BoolVarP(&showVersion, "version", "v", false, "print binary version")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "preview the command without running it")

	return cmd
}

func normalizeArgs(args []string) []string {
	normalized := make([]string, 0, len(args))

	for _, arg := range args {
		if converted, ok := normalizeLongFlag(arg); ok {
			normalized = append(normalized, converted)
		} else {
			normalized = append(normalized, arg)
		}
	}

	return normalized
}

func normalizeLongFlag(arg string) (string, bool) {
	switch {
	case arg == "-version", strings.HasPrefix(arg, "-version="):
		return "-" + arg, true
	case arg == "-help", strings.HasPrefix(arg, "-help="):
		return "-" + arg, true
	case arg == "-dry-run", strings.HasPrefix(arg, "-dry-run="):
		return "-" + arg, true
	default:
		return "", false
	}
}
