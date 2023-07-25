package alpine

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/reproducible-containers/repro-get/pkg/apkutil"
	"github.com/reproducible-containers/repro-get/pkg/cache"
	"github.com/reproducible-containers/repro-get/pkg/distro"
	"github.com/reproducible-containers/repro-get/pkg/filespec"
	"github.com/reproducible-containers/repro-get/pkg/urlopener"
	"github.com/sirupsen/logrus"
)

const Name = "alpine"

func New() distro.Distro {
	d := &alpine{
		info: distro.Info{
			Name: Name,
			DefaultProviders: []string{
				"https://dl-cdn.alpinelinux.org/alpine/{{.Name}}",
			},
			Experimental:                   true,
			CacheIsNeededForGeneratingHash: true,
		},
	}
	return d
}

type alpine struct {
	info      distro.Info
	installed map[string]apkutil.APK
}

func (d *alpine) Info() distro.Info {
	return d.info
}

func (d *alpine) GenerateHash(ctx context.Context, hw distro.HashWriter, opts distro.HashOpts) error {
	if opts.Cache == nil {
		return errors.New("cache is required")
	}
	names := opts.FilterByName
	if len(names) == 0 {
		apks, err := Installed()
		if err != nil {
			return err
		}
		if len(apks) == 0 {
			return errors.New("no package is installed?")
		}
		for name := range apks {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	dummyDir, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dummyDir)
	urlsCmd := exec.CommandContext(ctx, "apk", append([]string{"fetch", "--simulate", "--output=" + dummyDir, "--url"}, names...)...)
	urlsCmd.Stderr = os.Stderr
	urls, err := urlsCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute %v: %w", urlsCmd.Args, err)
	}
	return d.generateHashWithURLReader(ctx, hw, opts.Cache, bytes.NewReader(urls))
}

func (d *alpine) generateHashWithURLReader(ctx context.Context, hw distro.HashWriter, c *cache.Cache, r io.Reader) error {
	sc := bufio.NewScanner(r)
	urlOpener := urlopener.New()
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		u, err := url.Parse(trimmed)
		if err != nil {
			return err
		}
		if err := d.generateHashWithURL(ctx, hw, c, urlOpener, u); err != nil {
			return err
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return nil
}

func (d *alpine) generateHashWithURL(ctx context.Context, hw distro.HashWriter, c *cache.Cache, urlOpener *urlopener.URLOpener, u *url.URL) error {
	logrus.Debugf("Generating the hash for %q", u.Redacted())
	if u.Scheme != "https" {
		return fmt.Errorf("expected an https url, got %q", u.Redacted())
	}
	fname, err := urlToFilenameWithoutProvider(u)
	if err != nil {
		return err
	}
	basename := path.Base(fname)
	if sha256sum, err := c.SHA256ByOriginURL(u); err == nil {
		logrus.Debugf("%q: found cached sha256sum %s for %q", basename, sha256sum, u.Redacted())
		return hw(sha256sum, fname)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to check the cached sha256 by URL %q: %w", u.Redacted(), err)
	}
	logrus.Debugf("%q: downloading from %q", basename, u.Redacted())
	m := &cache.Metadata{
		Basename: basename,
	}
	sha256sum, err := c.ImportWithURL(u, m)
	if err != nil {
		return err
	}
	return hw(sha256sum, fname)
}

// urlToFilenameWithoutProvider converts
// "https://dl-cdn.alpinelinux.org/alpine/v3.16/main/x86_64/ca-certificates-bundle-20220614-r0.apk"
// to
// "v3.16/main/x86_64/ca-certificates-bundle-20220614-r0.apk"
func urlToFilenameWithoutProvider(u *url.URL) (string, error) {
	sp := strings.Split(u.Path, "/")
	for i := range sp {
		if i >= 1 && strings.HasPrefix(sp[i-1], "alpine") && sp[i][0] == 'v' && '1' <= sp[i][1] && sp[i][1] <= '9' {
			return strings.Join(sp[i:], "/"), nil
		}
	}
	return "", fmt.Errorf("failed to parse %q", u.Redacted())
}

func (d *alpine) InspectFile(ctx context.Context, sp filespec.FileSpec, opts distro.InspectFileOpts) (*distro.FileInfo, error) {
	inf := &distro.FileInfo{
		FileSpec: sp,
	}
	if sp.APK == nil {
		return inf, nil
	}
	inf.IsPackage = true
	inf.PackageName = sp.APK.Package
	if opts.CheckInstalled {
		if d.installed == nil {
			var err error
			d.installed, err = Installed()
			if err != nil {
				return inf, fmt.Errorf("failed to detect installed packages: %w", err)
			}
		}
		k := sp.APK.Package
		if inst, ok := d.installed[k]; ok {
			installed := inst.Version == sp.APK.Version
			inf.Installed = &installed
		}
	}
	return inf, nil
}

// Installed returns the package map.
// The map key is the package name.
func Installed() (map[string]apkutil.APK, error) {
	cmd := exec.Command("apk", "info", "-v")
	cmd.Stderr = os.Stderr
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	// logrus.Debugf("Running %v", cmd.Args)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start %v: %w", cmd.Args, err)
	}
	return installed(r)
}

func installed(r io.Reader) (map[string]apkutil.APK, error) {
	pkgs := make(map[string]apkutil.APK)
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		pkg, err := apkutil.Split(trimmed)
		if err != nil {
			return nil, fmt.Errorf("failed to split %q into the package name and the version string: %w", trimmed, err)
		}
		pkgs[pkg.Package] = *pkg
	}
	return pkgs, sc.Err()
}

func (d *alpine) InstallPackages(ctx context.Context, c *cache.Cache, pkgs []filespec.FileSpec, opts distro.InstallOpts) error {
	if len(pkgs) == 0 {
		return nil
	}
	cmdName, err := exec.LookPath("apk")
	if err != nil {
		return err
	}
	tmpDir, err := os.MkdirTemp("", "repro-get-apk-*.tmp")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	args := []string{"add", "--no-network"}
	logrus.Infof("Running '%s %s ...' with %d packages", cmdName, strings.Join(args, " "), len(pkgs))
	for _, pkg := range pkgs {
		blob, err := c.BlobAbsPath(pkg.SHA256)
		if err != nil {
			return err
		}
		ln, err := securejoin.SecureJoin(tmpDir, pkg.Basename)
		if err != nil {
			return err
		}
		if err := os.Symlink(blob, ln); err != nil {
			return err
		}
		args = append(args, ln)
	}
	cmd := exec.CommandContext(ctx, cmdName, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	logrus.Debugf("Running %v", cmd.Args)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (d *alpine) GenerateDockerfile(ctx context.Context, dir string, args distro.DockerfileTemplateArgs, opts distro.DockerfileOpts) error {
	return distro.ErrNotImplemented
}
