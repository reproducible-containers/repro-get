package main

import (
	"github.com/reproducible-containers/repro-get/pkg/archutil"
	"github.com/reproducible-containers/repro-get/pkg/cache"
	"github.com/reproducible-containers/repro-get/pkg/downloader"
	"github.com/reproducible-containers/repro-get/pkg/filespec"
	"github.com/spf13/cobra"
)

func newDownloadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download [SHA256SUMS]...",
		Short: "Download packages into the cache",
		Long: `Download packages into the cache.
Use 'repro-get cache export' for exporting the cache.`,
		Example: "  repro-get download SHA256SUMS-" + archutil.OCIArchDashVariant(),
		Args:    cobra.MinimumNArgs(1),
		RunE:    downloadAction,

		DisableFlagsInUseLine: true,
	}

	return cmd
}

func downloadAction(cmd *cobra.Command, args []string) error {
	d, err := getDistro(cmd)
	if err != nil {
		return err
	}

	opts := downloader.Opts{
		SkipInstalled: false,
	}

	ctx := cmd.Context()
	flags := cmd.Flags()
	cacheStr, err := flags.GetString("cache")
	if err != nil {
		return err
	}
	cache, err := cache.New(cacheStr)
	if err != nil {
		return err
	}
	opts.Providers, err = flags.GetStringSlice("provider")
	if err != nil {
		return err
	}

	fileSpecs, err := filespec.NewFromSHA256SUMSFiles(args...)
	if err != nil {
		return err
	}

	_, err = downloader.Download(ctx, d, cache, fileSpecs, opts)
	return err
}
