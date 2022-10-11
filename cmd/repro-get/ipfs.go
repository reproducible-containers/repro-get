package main

import (
	"github.com/spf13/cobra"
)

func newIPFSCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ipfs",
		Short:         "Manage IPFS",
		Args:          cobra.NoArgs,
		RunE:          needsSubcommand,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(
		newIPFSPushCommand(),
	)
	return cmd
}
