package main

import (
	"github.com/spf13/cobra"
)

func newDockerfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "dockerfile",
		Short:         "Manage dockerfiles (EXPERIMENTAL)",
		Args:          cobra.NoArgs,
		RunE:          needsSubcommand,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(
		newDockerfileGenerateCommand(),
	)
	return cmd
}
