package distro

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/reproducible-containers/repro-get/pkg/cache"
	"github.com/reproducible-containers/repro-get/pkg/filespec"
	"github.com/sirupsen/logrus"
)

var ErrNotImplemented = errors.New("the specified distro driver does not implement the requested feature")

// Distro is a distro driver.
type Distro interface {
	// Info returns the distro driver info.
	Info() Info

	// GenerateHash generates hash files.
	GenerateHash(ctx context.Context, hw HashWriter, opts HashOpts) error

	// InspectFile inspects a file
	InspectFile(ctx context.Context, sp filespec.FileSpec, opts InspectFileOpts) (*FileInfo, error)

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

type FileInfo struct {
	filespec.FileSpec
	IsPackage   bool
	IsAux       bool
	PackageName string
	Installed   *bool
}

type InspectFileOpts struct {
	CheckInstalled bool // can be slow
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

	ReproGetVersion string // e.g., "v0.1.0"
	ReproGetSHASHA  string // sha256sum of SHA256SUMS, e.g., "a23ee0e0a2a2e940809b968befc84aa928323c86d3f4eef1f1653c96c2861632" for https://github.com/reproducible-containers/repro-get/releases/download/v0.1.0/SHA256SUMS
}

//go:embed distroutil/dockerfilesnippets/Dockerfile.fetch-repro-get.snippet
var dockerfileFetchReproGetSnippet string

var DockerfileTemplateFuncMap = template.FuncMap{
	"join": strings.Join,
	"snippet": func(name string) (string, error) {
		switch name {
		case "fetch-repro-get":
			return dockerfileFetchReproGetSnippet, nil
		}
		return "", fmt.Errorf("unknown snippet name %q", name)
	},
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
	AuxFiles []filespec.FileSpec
}
