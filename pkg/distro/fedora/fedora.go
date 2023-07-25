package fedora

import (
	"bufio"
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

	"github.com/reproducible-containers/repro-get/pkg/cache"
	"github.com/reproducible-containers/repro-get/pkg/distro"
	"github.com/reproducible-containers/repro-get/pkg/filespec"
	"github.com/reproducible-containers/repro-get/pkg/rpmutil"
	"github.com/reproducible-containers/repro-get/pkg/urlopener"
	"github.com/sirupsen/logrus"
)

const (
	Name         = "fedora"
	kojiPackages = "https://kojipkgs.fedoraproject.org/packages/"
)

func New() distro.Distro {
	d := &fedora{
		info: distro.Info{
			Name: Name,
			DefaultProviders: []string{
				kojiPackages + "{{.Name}}",
			},
			Experimental:                   true,
			CacheIsNeededForGeneratingHash: true,
		},
	}
	return d
}

type fedora struct {
	info      distro.Info
	installed map[string]rpmutil.RPM
}

func (d *fedora) Info() distro.Info {
	return d.info
}

func (d *fedora) GenerateHash(ctx context.Context, hw distro.HashWriter, opts distro.HashOpts) error {
	if opts.Cache == nil {
		return errors.New("cache is required")
	}
	names := opts.FilterByName
	if len(names) == 0 {
		rpms, err := Installed()
		if err != nil {
			return err
		}
		if len(rpms) == 0 {
			return errors.New("no package is installed?")
		}
		for _, rpm := range rpms {
			names = append(names, rpm.Package)
		}
	}
	sort.Strings(names)
	cmd := exec.CommandContext(ctx, "rpm", append([]string{"-qa", "--queryformat", "%{NAME}-%{VERSION}-%{RELEASE}.%{ARCH}.rpm,%{SOURCERPM}\n"}, names...)...)
	// logrus.Debugf("Executing %v", cmd.Args)
	cmd.Stderr = os.Stderr
	r, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer r.Close()
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("failed to execute %v: %w", cmd.Args, err)
	}
	return d.generateHash(ctx, hw, opts.Cache, r)
}

func (d *fedora) generateHash(ctx context.Context, hw distro.HashWriter, c *cache.Cache, r io.Reader) error {
	const expectedFields = 2
	sc := bufio.NewScanner(r)
	urlOpener := urlopener.New()
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		logrus.Debugf("Parsing <RPM>,<SRPM> line %q", trimmed)
		fields := strings.SplitN(trimmed, ",", expectedFields)
		if len(fields) != expectedFields {
			return fmt.Errorf("unexpected line %q: expected %d fields, got %d", line, expectedFields, len(fields))
		}
		rpmName := fields[0]
		srpmName := fields[1] // "(none)" for gpg-pubkey
		rpm, err := rpmutil.ParseFilename(rpmName)
		if err != nil {
			logrus.WithError(err).Warningf("Failed to parse the RPM name %q", rpmName)
			continue
		}
		if !strings.HasSuffix(srpmName, ".rpm") {
			logrus.Warningf("Failed to determine the source RPM name of the package %q: %q", rpmName, srpmName)
			continue
		}
		srpm, err := rpmutil.ParseFilename(srpmName)
		if err != nil {
			logrus.WithError(err).Warningf("Failed to parse the source RPM name %q (package %q)", srpmName, rpmName)
			continue
		}
		fname := fmt.Sprintf("%s/%s/%s/%s/%s", srpm.Package, srpm.Version, srpm.Release, rpm.Architecture, rpmName)
		if err := d.generateHash1(ctx, hw, c, urlOpener, fname); err != nil {
			return err
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return nil
}

func (d *fedora) generateHash1(ctx context.Context, hw distro.HashWriter, c *cache.Cache, urlOpener *urlopener.URLOpener, fname string) error {
	rawURL := kojiPackages + fname
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	logrus.Debugf("Generating the hash for %q", u.Redacted())
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

func (d *fedora) InspectFile(ctx context.Context, sp filespec.FileSpec, opts distro.InspectFileOpts) (*distro.FileInfo, error) {
	inf := &distro.FileInfo{
		FileSpec: sp,
	}
	if sp.RPM == nil {
		return inf, nil
	}
	inf.IsPackage = true
	inf.PackageName = sp.RPM.Package
	if opts.CheckInstalled {
		if d.installed == nil {
			var err error
			d.installed, err = Installed()
			if err != nil {
				return inf, fmt.Errorf("failed to detect installed packages: %w", err)
			}
		}
		k := sp.RPM.Package
		if sp.RPM.Architecture != "" {
			k += ":" + sp.RPM.Architecture
		}
		if inst, ok := d.installed[k]; ok {
			installed := inst.Version+"."+inst.Release == sp.RPM.Version+"."+sp.RPM.Release
			inf.Installed = &installed
		}
	}
	return inf, nil
}

// Installed returns the package map.
// The map key is Package + ":" + Architecture (if Architecture != "").
func Installed() (map[string]rpmutil.RPM, error) {
	cmd := exec.Command("rpm", "-qa")
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

func installed(r io.Reader) (map[string]rpmutil.RPM, error) {
	pkgs := make(map[string]rpmutil.RPM)
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		pkg, err := rpmutil.Split(trimmed)
		if err != nil {
			return pkgs, fmt.Errorf("failed to parse package string %q: %w", trimmed, err)
		}
		k := pkg.Package
		if pkg.Architecture != "" {
			k += ":" + pkg.Architecture
		}
		pkgs[k] = *pkg
	}
	return pkgs, sc.Err()
}

func (d *fedora) checkSigs(ctx context.Context, c *cache.Cache, pkgs []filespec.FileSpec, opts distro.InstallOpts) error {
	if len(pkgs) == 0 {
		return nil
	}
	cmdName, err := exec.LookPath("rpmkeys")
	if err != nil {
		return err
	}
	args := []string{"--checksig"}
	logrus.Infof("Running '%s %s ...' with %d packages", cmdName, strings.Join(args, " "), len(pkgs))
	for _, pkg := range pkgs {
		blob, err := c.BlobAbsPath(pkg.SHA256)
		if err != nil {
			return err
		}
		args = append(args, blob)
	}
	cmd := exec.CommandContext(ctx, cmdName, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	logrus.Debugf("Running %v", cmd.Args)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (d *fedora) InstallPackages(ctx context.Context, c *cache.Cache, pkgs []filespec.FileSpec, opts distro.InstallOpts) error {
	if len(pkgs) == 0 {
		return nil
	}
	if err := d.checkSigs(ctx, c, pkgs, opts); err != nil {
		return fmt.Errorf("failed to check the RPM signatures: %w", err)
	}
	cmdName, err := exec.LookPath("rpm")
	if err != nil {
		return err
	}
	args := []string{"-Uvh"}
	logrus.Infof("Running '%s %s ...' with %d packages", cmdName, strings.Join(args, " "), len(pkgs))
	for _, pkg := range pkgs {
		blob, err := c.BlobAbsPath(pkg.SHA256)
		if err != nil {
			return err
		}
		args = append(args, blob)
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

func (d *fedora) GenerateDockerfile(ctx context.Context, dir string, args distro.DockerfileTemplateArgs, opts distro.DockerfileOpts) error {
	return distro.ErrNotImplemented
}
