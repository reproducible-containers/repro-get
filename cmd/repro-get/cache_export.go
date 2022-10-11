package main

import (
	"github.com/reproducible-containers/repro-get/pkg/cache"
	"github.com/reproducible-containers/repro-get/pkg/distro"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newCacheExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "export DIR",
		Short:   "Export the cached package files to the specified dir",
		Example: "  repro-get cache export .",
		Args:    cobra.ExactArgs(1),
		RunE:    cacheExportAction,

		DisableFlagsInUseLine: true,
	}
	return cmd
}

func cacheExportAction(cmd *cobra.Command, args []string) error {
	dir := args[0]
	w := cmd.OutOrStdout()
	hw := distro.NewHashWriter(w)
	flags := cmd.Flags()
	cacheStr, err := flags.GetString("cache")
	if err != nil {
		return err
	}
	cache, err := cache.New(cacheStr)
	if err != nil {
		return err
	}
	exported, err := cache.Export(dir)
	for basename, sha256sum := range exported {
		if hwErr := hw(sha256sum, basename); hwErr != nil {
			logrus.Warn(hwErr)
		}
	}
	return err
}
