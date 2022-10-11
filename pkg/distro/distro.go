package distro

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/reproducible-containers/repro-get/pkg/cache"
	"github.com/reproducible-containers/repro-get/pkg/filespec"
	"github.com/sirupsen/logrus"
)

// Distro is a distro driver.
type Distro interface {
	// Info returns the distro driver info.
	Info() Info

	// GenerateHash generates hash files.
	GenerateHash(ctx context.Context, hw HashWriter, opts HashOpts) error

	// PackageName returns the package name string that can be used in HashOpts.FilterByName.
	PackageName(sp filespec.FileSpec) (string, error)

	// IsPackageVersionInstalled checks that the package with the specified version is installed.
	IsPackageVersionInstalled(ctx context.Context, sp filespec.FileSpec) (bool, error)

	// InstallPackages installs the packages. The packages must be cached.
	InstallPackages(ctx context.Context, c *cache.Cache, pkgs []filespec.FileSpec, opts InstallOpts) error

	// GenerateDockerfile generates dockerfiles.
	GenerateDockerfile(ctx context.Context, dir string, args DockerfileTemplateArgs, opts DockerfileOpts) error
}

type Info struct {
	Name                           string   `json:"Name"` // "debian", "ubuntu", ...
	DefaultProviders               []string `json:"DefaultProviders"`
	Experimental                   bool     `json:"Experimental"`
	CacheIsNeededForGeneratingHash bool     `json:"-"` // Implementation detail, not exposed in the JSON
}

type HashOpts struct {
	FilterByName []string     // No filter when empty
	Cache        *cache.Cache // Used only if Info.CacheIsNeededForGeneratingHash is true
}

type HashWriter func(sha256sum, filename string) error

func NewHashWriter(w io.Writer) HashWriter {
	return func(sha256sum, filename string) error {
		_, err := fmt.Fprintln(w, sha256sum+"  "+filename)
		return err
	}
}

type DockerfileTemplateArgs struct {
	BaseImage          string
	BaseImageOrig      string
	Packages           []string
	OCIArchDashVariant string
	Providers          []string
}

var DockerfileTemplateFuncMap = template.FuncMap{
	"join": strings.Join,
}

func (a *DockerfileTemplateArgs) WriteToFile(f, tmpl string) error {
	logrus.Infof("Generating %q", f)
	parsed, err := template.New(filepath.Base(f)).Funcs(DockerfileTemplateFuncMap).Parse(tmpl)
	if err != nil {
		return err
	}
	var b bytes.Buffer
	if err = parsed.Execute(&b, a); err != nil {
		return err
	}
	return os.WriteFile(f, b.Bytes(), 0644)
}

type DockerfileOpts struct {
	GenerateHash bool
}

type InstallOpts struct {
	// Reserved
}
