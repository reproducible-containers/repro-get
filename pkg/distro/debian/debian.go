package debian

import (
	"bufio"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/reproducible-containers/repro-get/pkg/cache"
	"github.com/reproducible-containers/repro-get/pkg/distro"
	"github.com/reproducible-containers/repro-get/pkg/dpkgutil"
	"github.com/reproducible-containers/repro-get/pkg/filespec"
	"github.com/sirupsen/logrus"
	"pault.ag/go/debian/control"
	"pault.ag/go/debian/version"
)

const (
	NameDebian = "debian"
	NameUbuntu = "ubuntu"
	Name       = NameDebian
)

func New() distro.Distro {
	d := &debian{
		info: distro.Info{
			Name: NameDebian,
			DefaultProviders: []string{
				// HTTPS is not used by default in the apt-get ecosystem. See also README.md.
				//
				// deb.debian.org: multi-arch, ephemeral
				"http://deb.debian.org/debian/{{.Name}}",
				"http://deb.debian.org/debian-security/{{.Name}}",
				//
				// snapshot-cloudflare.debian.org: multi-arch, persistent, slow, experimental (?)
				"http://snapshot-cloudflare.debian.org/archive/debian/{{timeToDebianSnapshot .Epoch}}/{{.Name}}",
				"http://snapshot-cloudflare.debian.org/archive/debian-security/{{timeToDebianSnapshot .Epoch}}/{{.Name}}",
				//
				// snapshot.debian.org: multi-arch, persistent, very slow
				"http://snapshot.debian.org/archive/debian/{{timeToDebianSnapshot .Epoch}}/{{.Name}}",
				"http://snapshot.debian.org/archive/debian-security/{{timeToDebianSnapshot .Epoch}}/{{.Name}}",
				//
				// archive.debian.org: multi-arch, persistent, EOL only
				"http://archive.debian.org/debian/{{.Name}}",
				"http://archive.debian.org/debian-security/{{.Name}}",
				//
				// debian.notset.fr: slow and amd64 only, but accessible by SHA256
				// Down as of June 2023: https://github.com/fepitre/debian-snapshot/issues/20
				"http://debian.notset.fr/snapshot/by-hash/SHA256/{{.SHA256}}",
			},
		},
	}
	return d
}

func NewUbuntu() distro.Distro {
	d := &debian{
		info: distro.Info{
			Name: NameUbuntu,
			DefaultProviders: []string{
				// HTTPS is not used by default in the apt-get ecosystem. See also README.md.
				"http://ports.ubuntu.com/{{.Name}}",                                 // multi-arch, ephemeral
				"http://launchpad.net/ubuntu/+archive/primary/+files/{{.Basename}}", // multi-arch, persistent
				"http://archive.ubuntu.com/ubuntu/{{.Name}}",                        // amd64 only, ephemeral
				"http://old-releases.ubuntu.com/ubuntu/{{.Name}}",                   // multi-arch, persistent, EOL only
			},
		},
	}
	return d
}

type debian struct {
	info      distro.Info
	installed map[string]dpkgutil.Dpkg
}

func (d *debian) Info() distro.Info {
	return d.info
}

func (d *debian) GenerateHash(ctx context.Context, hw distro.HashWriter, opts distro.HashOpts) error {
	names := opts.FilterByName
	if len(names) == 0 {
		dpkgs, err := Installed()
		if err != nil {
			return err
		}
		if len(dpkgs) == 0 {
			return errors.New("no package is installed?")
		}
		for name := range dpkgs {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	// /var/lib/dpkg/available is only updated by dselect,
	// so we have to shell out `apt-cache show PKGS...`
	aptCacheArgs := append([]string{"show"}, names...)
	aptCacheCmd := exec.Command("apt-cache", aptCacheArgs...)
	aptCacheCmd.Stderr = os.Stderr
	aptCacheR, err := aptCacheCmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer aptCacheR.Close()
	// logrus.Debugf("Running %v", aptCacheCmd.Args)
	if err := aptCacheCmd.Start(); err != nil {
		return fmt.Errorf("failed to start %v: %w", aptCacheCmd.Args, err)
	}
	if err = generateHash(hw, aptCacheR); err != nil {
		return fmt.Errorf("failed to parse the output of %v: %w", aptCacheCmd.Args, err)
	}
	return nil
}

func generateHash(hw distro.HashWriter, r io.Reader) error {
	bufR := bufio.NewReader(r)

	var paragraphs []control.BinaryParagraph
	if err := control.Unmarshal(&paragraphs, bufR); err != nil {
		return err
	}
	// logrus.Debugf("Scanning %d entries", len(paragraphs))

	seen := make(map[string]string)
	for _, f := range paragraphs {
		ver := f.Paragraph.Values["Version"]
		seenK := f.Package + ":" + f.Paragraph.Values["Architecture"]
		if seenV, ok := seen[seenK]; ok {
			seenVParsed, err := version.Parse(seenV)
			if err != nil {
				logrus.WithError(err).Warnf("Failed to parse version %q", seenV)
				continue
			}
			verParsed, err := version.Parse(ver)
			if err != nil {
				logrus.WithError(err).Warnf("Failed to parse version %q", ver)
				continue
			}
			if version.Compare(seenVParsed, verParsed) > 0 {
				continue
			}
		}
		seen[seenK] = ver
		dpkgFilename := f.Paragraph.Values["Filename"]
		if dpkgFilename == "" {
			logrus.Warnf("No Filename found for package %q (Hint: try 'apt-get update')", f.Package)
			continue
		}

		sha256Digest := f.Paragraph.Values["SHA256"]
		if sha256Digest == "" {
			logrus.Warnf("No SHA256 found for package %q (Hint: try 'apt-get update')", f.Package)
			continue
		}
		if err := hw(sha256Digest, dpkgFilename); err != nil {
			return err
		}
	}
	return nil
}

func (d *debian) InspectFile(ctx context.Context, sp filespec.FileSpec, opts distro.InspectFileOpts) (*distro.FileInfo, error) {
	inf := &distro.FileInfo{
		FileSpec: sp,
	}
	if sp.Dpkg == nil {
		return inf, nil
	}
	inf.IsPackage = true
	inf.PackageName = sp.Dpkg.Package
	if opts.CheckInstalled {
		if d.installed == nil {
			var err error
			d.installed, err = Installed()
			if err != nil {
				return inf, fmt.Errorf("failed to detect installed packages: %w", err)
			}
		}
		k := sp.Dpkg.Package
		if sp.Dpkg.Architecture != "" {
			k += ":" + sp.Dpkg.Architecture
		}
		if inst, ok := d.installed[k]; ok {
			installed := inst.Version == sp.Dpkg.Version
			inf.Installed = &installed
		}
	}
	return inf, nil
}

// Installed returns the package map.
// The map key is Package + ":" + Architecture (if Architecture != "").
func Installed() (map[string]dpkgutil.Dpkg, error) {
	cmd := exec.Command("dpkg-query", "-f", "${Package},${Version},${Architecture}\n", "-W")
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

func installed(r io.Reader) (map[string]dpkgutil.Dpkg, error) {
	const expectedFields = 3
	pkgs := make(map[string]dpkgutil.Dpkg)
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		fields := strings.SplitN(trimmed, ",", expectedFields)
		if len(fields) != expectedFields {
			return pkgs, fmt.Errorf("unexpected line %q: expected %d fields, got %d", line, expectedFields, len(fields))
		}
		pkg := dpkgutil.Dpkg{
			Package:      fields[0],
			Version:      fields[1],
			Architecture: fields[2],
		}
		k := pkg.Package
		if pkg.Architecture != "" {
			k += ":" + pkg.Architecture
		}
		pkgs[k] = pkg
	}
	return pkgs, sc.Err()
}

func (d *debian) InstallPackages(ctx context.Context, c *cache.Cache, pkgs []filespec.FileSpec, opts distro.InstallOpts) error {
	if len(pkgs) == 0 {
		return nil
	}
	cmdName, err := exec.LookPath("dpkg")
	if err != nil {
		return err
	}
	args := []string{"-i"}
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

var (
	//go:embed Dockerfile.generate-hash.tmpl
	dockerfileGenerateHashTmpl string

	//go:embed Dockerfile.tmpl
	dockerfileTmpl string
)

func (d *debian) GenerateDockerfile(ctx context.Context, dir string, args distro.DockerfileTemplateArgs, opts distro.DockerfileOpts) error {
	if d.info.Name != NameDebian {
		return fmt.Errorf("generating dockerfiles needs the distro driver to be set to %q, not %q", NameDebian, d.info.Name)
	}
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
