package arch

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
	"github.com/reproducible-containers/repro-get/pkg/pacmanutil"
	"github.com/reproducible-containers/repro-get/pkg/urlopener"
	"github.com/sirupsen/logrus"
)

const (
	Name = "arch"
)

func New() distro.Distro {
	d := &arch{
		info: distro.Info{
			Name: Name,
			DefaultProviders: []string{
				"https://archive.archlinux.org/packages/{{.Name}}",
			},
			Experimental:                   true,
			CacheIsNeededForGeneratingHash: true,
		},
	}
	return d
}

type arch struct {
	info      distro.Info
	installed map[string]pacmanutil.Pacman
}

func (d *arch) Info() distro.Info {
	return d.info
}

func (d *arch) GenerateHash(ctx context.Context, hw distro.HashWriter, opts distro.HashOpts) error {
	if opts.Cache == nil {
		return errors.New("cache is required")
	}
	names := opts.FilterByName
	if len(names) == 0 {
		pkgs, err := Installed()
		if err != nil {
			return err
		}
		if len(pkgs) == 0 {
			return errors.New("no package is installed?")
		}
		for _, pkg := range pkgs {
			names = append(names, pkg.Package)
		}
	}
	sort.Strings(names)
	cmd := exec.CommandContext(ctx, "pacman", append([]string{"-Sddp"}, names...)...)
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

func (d *arch) generateHash(ctx context.Context, hw distro.HashWriter, c *cache.Cache, r io.Reader) error {
	sc := bufio.NewScanner(r)
	urlOpener := urlopener.New()
	for sc.Scan() {
		line := sc.Text()
		rawURL := strings.TrimSpace(line)
		if err := d.generateHash1(ctx, hw, c, urlOpener, rawURL); err != nil {
			return err
		}
		// TODO: download ".sig" too
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return nil
}

func (d *arch) generateHash1(ctx context.Context, hw distro.HashWriter, c *cache.Cache, uo *urlopener.URLOpener, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	logrus.Debugf("Generating the hash for %q", u.Redacted())
	basename := path.Base(u.Path)
	pkg, err := pacmanutil.ParseFilename(basename)
	if err != nil {
		return err
	}
	fname := fmt.Sprintf("%c/%s/%s", pkg.Package[0], pkg.Package, basename)
	if sha256sum, err := c.SHA256ByOriginURL(u); err == nil {
		logrus.Debugf("%q: found cached sha256sum %s for %q", basename, sha256sum, u.Redacted())
		return hw(sha256sum, fname)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to check the cached sha256 by URL %q: %w", u.Redacted(), err)
	}
	logrus.Debugf("%q: downloading from %q", basename, u.Redacted())
	sha256sum, err := c.ImportWithURL(u)
	if err != nil {
		return err
	}
	return hw(sha256sum, fname)
}

func (d *arch) PackageName(sp filespec.FileSpec) (string, error) {
	if sp.Pacman == nil {
		return "", fmt.Errorf("pacman information not available for %q", sp.Name)
	}
	return sp.Pacman.Package, nil
}

func (d *arch) IsPackageVersionInstalled(ctx context.Context, sp filespec.FileSpec) (bool, error) {
	if sp.Pacman == nil {
		return false, fmt.Errorf("pacman information not available for %q", sp.Name)
	}
	if d.installed == nil {
		var err error
		d.installed, err = Installed()
		if err != nil {
			return false, fmt.Errorf("failed to detect installed rpms: %w", err)
		}
	}
	k := sp.Pacman.Package
	if sp.Pacman.Architecture != "" {
		k += ":" + sp.Pacman.Architecture
	}
	inst, ok := d.installed[k]
	if !ok {
		return false, nil
	}
	return inst.Version == sp.Pacman.Version, nil
}

// Installed returns the package map.
// The map key is Package + ":" + Architecture (if Architecture != "").
func Installed() (map[string]pacmanutil.Pacman, error) {
	cmd := exec.Command("pacman", "-Qi")
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

func installed(r io.Reader) (map[string]pacmanutil.Pacman, error) {
	pkgs := make(map[string]pacmanutil.Pacman)
	sc := bufio.NewScanner(r)
	var pkg pacmanutil.Pacman
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			// Store the entry
			if k := pkg.Package; k != "" {
				if pkg.Architecture != "" {
					k += ":" + pkg.Architecture
				}
				pkgs[k] = pkg
			}
			continue
		}
		fields := strings.SplitN(trimmed, ":", 2)
		if len(fields) != 2 {
			// continuation of the previous line
			continue
		}
		propK, propV := strings.TrimSpace(fields[0]), strings.TrimSpace(fields[1])
		switch propK {
		case "Name":
			pkg.Package = propV
			pkg.Version = ""
			pkg.Architecture = ""
		case "Version":
			pkg.Version = propV
		case "Architecture":
			pkg.Architecture = propV
		}
	}
	return pkgs, sc.Err()
}

func (d *arch) InstallPackages(ctx context.Context, c *cache.Cache, pkgs []filespec.FileSpec, opts distro.InstallOpts) error {
	if len(pkgs) == 0 {
		return nil
	}
	cmdName, err := exec.LookPath("pacman")
	if err != nil {
		return err
	}
	args := []string{"-Uv", "--noconfirm"}
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

func (d *arch) GenerateDockerfile(ctx context.Context, dir string, args distro.DockerfileTemplateArgs, opts distro.DockerfileOpts) error {
	return distro.ErrNotImplemented
}
