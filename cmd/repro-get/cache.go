package main

import (
	"github.com/spf13/cobra"
)

func newCacheCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "cache",
		Short:         "Manage cache",
		Args:          cobra.NoArgs,
		RunE:          needsSubcommand,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(
		newCacheImportCommand(),
		newCacheExportCommand(),
		newCacheCleanCommand(),
	)
	return cmd
}
