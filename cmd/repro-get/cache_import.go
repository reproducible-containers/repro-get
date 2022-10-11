package main

import (
	"github.com/reproducible-containers/repro-get/pkg/cache"
	"github.com/reproducible-containers/repro-get/pkg/distro"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newCacheImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "import [FILES]...",
		Short:   "Import package files into the cache",
		Example: "  repro-get cache import *.dpkg",
		Args:    cobra.MinimumNArgs(1),
		RunE:    cacheImportAction,

		DisableFlagsInUseLine: true,
	}
	return cmd
}

func cacheImportAction(cmd *cobra.Command, args []string) error {
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
	imported, err := cache.Import(args...)
	for basename, sha256sum := range imported {
		if hwErr := hw(sha256sum, basename); hwErr != nil {
			logrus.Warn(hwErr)
		}
	}
	return err
}
