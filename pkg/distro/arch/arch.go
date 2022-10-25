package arch

import (
	"bufio"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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
		sigRawURL := rawURL + ".sig"
		if err := d.generateHash1(ctx, hw, c, urlOpener, sigRawURL); err != nil {
			return err
		}
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
	pkg, err := pacmanutil.ParseFilename(strings.TrimSuffix(basename, ".sig"))
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

func (d *arch) InspectFile(ctx context.Context, sp filespec.FileSpec, opts distro.InspectFileOpts) (*distro.FileInfo, error) {
	inf := &distro.FileInfo{
		FileSpec: sp,
	}
	var pkg pacmanutil.Pacman
	if sp.Pacman == nil {
		if strings.HasSuffix(sp.Name, ".pkg.tar.zst.sig") {
			inf.IsAux = true
			pkgP, err := pacmanutil.Split(strings.TrimSuffix(sp.Name, ".pkg.tar.zst.dig"))
			if err != nil {
				return inf, err
			}
			pkg = *pkgP
		} else {
			return inf, nil
		}
	} else {
		inf.IsPackage = true
		pkg = *sp.Pacman
	}
	inf.PackageName = pkg.Package
	if opts.CheckInstalled {
		if d.installed == nil {
			var err error
			d.installed, err = Installed()
			if err != nil {
				return inf, fmt.Errorf("failed to detect installed packages: %w", err)
			}
		}
		k := pkg.Package
		if pkg.Architecture != "" {
			k += ":" + pkg.Architecture
		}
		if inst, ok := d.installed[k]; ok {
			installed := inst.Version == pkg.Version
			inf.Installed = &installed
		}
	}
	return inf, nil
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

	tmpDir, err := os.MkdirTemp("", "repro-get-pacman-*.tmp")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// Prepare symlinks
	for _, f := range append(pkgs, opts.AuxFiles...) {
		blob, err := c.BlobAbsPath(f.SHA256)
		if err != nil {
			return err
		}
		if filepath.Base(f.Basename) != f.Basename {
			return fmt.Errorf("bad basename %q", f.Basename)
		}
		ln := filepath.Join(tmpDir, f.Basename) // no need to use securejoin (f.Basename is verified)
		if err := os.Symlink(blob, ln); err != nil {
			return err
		}
	}

	cmdName, err := exec.LookPath("pacman-key")
	if err != nil {
		return err
	}
	logrus.Infof("Running '%s --verify ...' with %d signatures", cmdName, len(opts.AuxFiles))

	pkgSigMap := make(map[string]string) // key: pkg basename, val: sig basename
	for _, f := range opts.AuxFiles {
		if !strings.HasSuffix(f.Basename, ".sig") {
			return fmt.Errorf("expected *.sig, got %q", f.Basename)
		}
		pkgSigMap[strings.TrimSuffix(f.Basename, ".sig")] = f.Basename
		file := filepath.Join(tmpDir, f.Basename) // securejoin can't be used for symlinks; f.Basename is verified
		args := []string{"--verify", file}

		cmd := exec.CommandContext(ctx, cmdName, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		logrus.Debugf("Running %v", cmd.Args)
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	cmdName, err = exec.LookPath("pacman")
	if err != nil {
		return err
	}
	args := []string{"-Uv", "--noconfirm"}
	logrus.Infof("Running '%s %s ...' with %d packages", cmdName, strings.Join(args, " "), len(pkgs))
	for _, f := range pkgs {
		if _, ok := pkgSigMap[f.Basename]; !ok {
			return fmt.Errorf("no signature found for package %q", f.Basename)
		}
		file := filepath.Join(tmpDir, f.Basename) // securejoin can't be used for symlinks; f.Basename is verified
		args = append(args, file)
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

var (
	//go:embed Dockerfile.generate-hash.tmpl
	dockerfileGenerateHashTmpl string

	//go:embed Dockerfile.tmpl
	dockerfileTmpl string
)

func (d *arch) GenerateDockerfile(ctx context.Context, dir string, args distro.DockerfileTemplateArgs, opts distro.DockerfileOpts) error {
	if opts.GenerateHash {
		f := filepath.Join(dir, "Dockerfile.generate-hash") // no need to use securejoin (const)
		if err := args.WriteToFile(f, dockerfileGenerateHashTmpl); err != nil {
			return fmt.Errorf("failed to generate %q: %w", f, err)
		}
	}
	f := filepath.Join(dir, "Dockerfile") // no need to use securejoin (const)
	if err := args.WriteToFile(f, dockerfileTmpl); err != nil {
		return fmt.Errorf("failed to generate %q: %w", f, err)
	}
	return nil
}
