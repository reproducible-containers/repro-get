package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newCacheCleanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clean DIR",
		Short:   "Clean the cache",
		Example: "  repro-get cache clean .",
		Args:    cobra.NoArgs,
		RunE:    cacheCleanAction,

		DisableFlagsInUseLine: true,
	}
	return cmd
}

func cacheCleanAction(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()
	cacheStr, err := flags.GetString("cache")
	if err != nil {
		return err
	}
	logrus.Infof("Removing %q", cacheStr)
	return os.RemoveAll(cacheStr)
}
