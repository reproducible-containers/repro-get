package main

import (
	"github.com/spf13/cobra"
)

func newHashCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "hash",
		Short:         "Manage hash",
		Args:          cobra.NoArgs,
		RunE:          needsSubcommand,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(
		newHashGenerateCommand(),
		newHashUpdateCommand(),
		newHashInspectCommand(),
	)
	return cmd
}
