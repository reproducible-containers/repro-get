package version

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"runtime/debug"
	"strconv"
	"text/template"

	"github.com/opencontainers/go-digest"
	"github.com/reproducible-containers/repro-get/pkg/urlopener"
)

// Variables can be fulfilled on compilation time: -ldflags="-X github.com/reproducible-containers/repro-get/pkg/version.Version=v0.1.2"
var (
	Version             string
	DownloadableVersion string // <= Version

	SHA256SUMSDownloadURLTemplate = "https://github.com/reproducible-containers/repro-get/releases/download/{{.DownloadableVersion}}/SHA256SUMS"
)

// SHASHA returns the sha256sum of the SHA256SUMS file.
func SHASHA(ctx context.Context, downloadableVersion string) (string, error) {
	tmpl, err := template.New("").Parse(SHA256SUMSDownloadURLTemplate)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	args := map[string]string{
		"DownloadableVersion": downloadableVersion,
	}
	if err = tmpl.Execute(&b, args); err != nil {
		return "", err
	}
	s := b.String()
	u, err := url.Parse(s)
	if err != nil {
		return "", err
	}
	uo := urlopener.New()
	r, _, err := uo.Open(ctx, u, "")
	if err != nil {
		return "", err
	}
	defer r.Close()
	digester := digest.SHA256.Digester()
	hasher := digester.Hash()
	if _, err = io.Copy(hasher, r); err != nil {
		return "", err
	}
	if err = r.Close(); err != nil {
		return "", err
	}
	return digester.Digest().Encoded(), nil
}

func GetVersion() string {
	if Version != "" {
		return Version
	}
	const unknown = "(unknown)"
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return unknown
	}
	// bi.Main.Version is always "(devel)" as of Go 1.19, but will change in the future:
	// https://github.com/golang/go/issues/50603#issuecomment-1076662671
	var (
		vcsRevision string
		vcsTime     string
		vcsModified bool
	)
	for _, f := range bi.Settings {
		switch f.Key {
		case "vcs.revision":
			vcsRevision = f.Value
		case "vcs.time":
			vcsTime = f.Value
		case "vcs.modified":
			vcsModified, _ = strconv.ParseBool(f.Value)
		}
	}
	if vcsRevision == "" {
		return unknown
	}
	v := vcsRevision
	if vcsModified {
		v += ".m"
	}
	if vcsTime != "" {
		v += " [" + vcsTime + "]"
	}
	return v
}
