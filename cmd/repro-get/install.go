package main

import (
	"github.com/reproducible-containers/repro-get/pkg/archutil"
	"github.com/reproducible-containers/repro-get/pkg/cache"
	"github.com/reproducible-containers/repro-get/pkg/distro"
	"github.com/reproducible-containers/repro-get/pkg/downloader"
	"github.com/reproducible-containers/repro-get/pkg/filespec"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newInstallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "install [flags] [SHA256SUMS]...",
		Short:   "Install packages with the hash file",
		Example: "  repro-get install SHA256SUMS-" + archutil.OCIArchDashVariant(),
		Args:    cobra.MinimumNArgs(1),
		RunE:    installAction,

		DisableFlagsInUseLine: true,
	}
	return cmd
}

func installAction(cmd *cobra.Command, args []string) error {
	d, err := getDistro(cmd)
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	flags := cmd.Flags()

	downloadOpts := downloader.Opts{
		SkipInstalled: true,
	}

	downloadOpts.Providers, err = flags.GetStringSlice("provider")
	if err != nil {
		return err
	}

	cacheStr, err := flags.GetString("cache")
	if err != nil {
		return err
	}
	cache, err := cache.New(cacheStr)
	if err != nil {
		return err
	}

	fileSpecs, err := filespec.NewFromSHA256SUMSFiles(args...)
	if err != nil {
		return err
	}

	downloadRes, err := downloader.Download(ctx, d, cache, fileSpecs, downloadOpts)
	if err != nil {
		return err
	}
	if len(downloadRes.PackagesToBeInstalled) == 0 {
		logrus.Info("No package to install")
		return nil
	}

	installOpts := distro.InstallOpts{
		AuxFiles: downloadRes.AuxFilesForInstallation,
	}
	return d.InstallPackages(ctx, cache, downloadRes.PackagesToBeInstalled, installOpts)
}
