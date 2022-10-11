package main

import (
	"encoding/json"
	"fmt"

	"github.com/reproducible-containers/repro-get/pkg/archutil"
	"github.com/reproducible-containers/repro-get/pkg/filespec"
	"github.com/spf13/cobra"
)

func newHashInspectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "inspect [SHA256SUMS]...",
		Short:   "Inspect the hash file",
		Example: "  repro-get hash inspect SHA256SUMS-" + archutil.OCIArchDashVariant(),
		Args:    cobra.MinimumNArgs(1),
		RunE:    hashInspectAction,

		DisableFlagsInUseLine: true,
	}

	return cmd
}

func hashInspectAction(cmd *cobra.Command, args []string) error {
	entries, err := filespec.NewFromSHA256SUMSFiles(args...)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(entries, "", "    ")
	if err != nil {
		return err
	}
	w := cmd.OutOrStdout()
	if _, err = fmt.Fprintln(w, string(b)); err != nil {
		return err
	}
	return nil
}
