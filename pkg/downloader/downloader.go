package downloader

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/fatih/color"
	"github.com/reproducible-containers/repro-get/pkg/cache"
	"github.com/reproducible-containers/repro-get/pkg/distro"
	"github.com/reproducible-containers/repro-get/pkg/filespec"
	"github.com/sirupsen/logrus"
)

type Result struct {
	PackagesToBeInstalled   []filespec.FileSpec // contains files that were already cached
	AuxFilesForInstallation []filespec.FileSpec
}

func (r *Result) keep(inf distro.FileInfo) {
	if inf.IsPackage {
		r.PackagesToBeInstalled = append(r.PackagesToBeInstalled, inf.FileSpec)
	}
	if inf.IsAux {
		r.AuxFilesForInstallation = append(r.AuxFilesForInstallation, inf.FileSpec)
	}
}

type Opts struct {
	Providers     []string
	SkipInstalled bool
}

func Download(ctx context.Context, d distro.Distro, cache *cache.Cache, fileSpecs map[string]*filespec.FileSpec, opts Opts) (*Result, error) {
	if d == nil {
		return nil, errors.New("distro driver needs to be specified")
	}
	if cache == nil {
		return nil, errors.New("cache needs to be specified")
	}

	providers := opts.Providers
	if len(providers) == 0 {
		providers = d.Info().DefaultProviders
	}
	if len(providers) == 0 {
		return nil, errors.New("provider needs to be specified")
	}

	var fnames []string
	for f := range fileSpecs {
		fnames = append(fnames, f)
	}
	sort.Strings(fnames)
	l := len(fnames)

	markUpProgressCounter := color.New(color.Bold).SprintFunc()
	markUpPackage := color.New(color.FgCyan).SprintFunc()
	markUpComment := color.New(color.FgHiBlack).SprintFunc()
	printPackageStatusBase := func(i int, pkg, s string, ff ...interface{}) {
		fmt.Println(markUpProgressCounter(fmt.Sprintf("(%03d/%03d)", i+1, l)) + " " + markUpPackage(pkg) + " " + markUpComment(fmt.Sprintf(s, ff...)))
	}

	var res Result
	for i, fname := range fnames {
		sp := fileSpecs[fname]
		printPackageStatus := func(s string, ff ...interface{}) {
			printPackageStatusBase(i, sp.Basename, s, ff...)
		}
		inf, err := d.InspectFile(ctx, *sp, distro.InspectFileOpts{})
		if err != nil {
			logrus.WithError(err).Warnf("Failed to inspect %+v", sp)
			continue
		}
		if !inf.IsPackage && !inf.IsAux {
			printPackageStatus("Not needed")
			continue
		}
		if opts.SkipInstalled {
			var installed bool
			infDeep, err := d.InspectFile(ctx, *sp, distro.InspectFileOpts{CheckInstalled: true})
			if err != nil {
				logrus.WithError(err).Warnf("Failed to check whether installed: %qw", sp.Basename)
			} else if infDeep.Installed != nil {
				installed = *infDeep.Installed
			}
			if installed {
				printPackageStatus("Already installed")
				continue
			}
		}
		cached, err := cache.Cached(sp.SHA256)
		if err != nil {
			logrus.WithError(err).Warnf("Failed to check whether %q (%q) is cached", sp.SHA256, sp.Basename)
			cached = false
		}
		if cached {
			printPackageStatus("Cached")
			res.keep(*inf)
			continue
		}
		for j, provider := range providers {
			u, err := sp.URL(provider)
			if err != nil {
				return nil, fmt.Errorf("failed to determine the URL of %v with the provider %q: %w", sp, provider, err)
			}
			printPackageStatus("Downloading from %s", u.Redacted())
			if err = cache.Ensure(ctx, u, sp.SHA256); err != nil {
				if j != len(providers)-1 {
					logrus.WithError(err).Warnf("Failed to download %s (%s), trying the next provider", sp.Basename, u.Redacted())
				} else {
					return nil, fmt.Errorf("failed to download %s (%s): %w", sp.Basename, u.Redacted(), err)
				}
			} else {
				break
			}
		}
		res.keep(*inf)
	}
	return &res, nil
}
