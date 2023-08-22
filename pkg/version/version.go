package version

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"text/template"

	"github.com/opencontainers/go-digest"
	"github.com/reproducible-containers/repro-get/pkg/urlopener"
	"github.com/sirupsen/logrus"
)

// Variables can be fulfilled on compilation time: -ldflags="-X github.com/reproducible-containers/repro-get/pkg/version.Version=v0.1.2"
var (
	Version string

	SHA256SUMSDownloadURLTemplate = "https://github.com/reproducible-containers/repro-get/releases/download/{{.Version}}/SHA256SUMS"
	LatestVersionURL              = "https://github.com/reproducible-containers/repro-get/releases/latest/download/VERSION"
)

const (
	Latest = "latest"
	Auto   = "auto" // current version or latest
)

type Downloadable struct {
	Version string
	SHASHA  string // the sha256sum of the SHA256SUMS file
}

func DetectDownloadable(ctx context.Context, version string) (*Downloadable, error) {
	var err error
	switch version {
	case Auto:
		currentVersion := GetVersion()
		rec, err := DetectDownloadable(ctx, currentVersion)
		if err != nil {
			logrus.WithError(err).Warnf("The current version %q does not seem downloadable, falling back to the latest version", currentVersion)
			rec, err = DetectDownloadable(ctx, Latest)
			if err != nil {
				return nil, fmt.Errorf("failed to detect the latest version: %w", err)
			}
		}
		logrus.Debugf("Automatically detected downloadable version: %+v", rec)
		return rec, nil
	case Latest:
		version, err = DetectLatest(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to detect the latest version: %w", err)
		}
		logrus.Infof("Detected latest version %q", version)
	default:
		// NOP
	}

	if !strings.HasPrefix(version, "v") || strings.Contains(version, "-g") || strings.HasSuffix(version, ".m") || len(version) > 30 {
		return nil, fmt.Errorf("non-downloadable version: %q", version)
	}
	d := &Downloadable{
		Version: version,
	}
	d.SHASHA, err = shasha(ctx, version)
	if err != nil {
		return nil, fmt.Errorf("failed to detect the sha256sum of the SHA256SUMS file for version %q: %w", version, err)
	}
	return d, nil
}

func DetectLatest(ctx context.Context) (string, error) {
	u, err := url.Parse(LatestVersionURL)
	if err != nil {
		return "", err
	}
	uo := urlopener.New()
	r, _, err := uo.Open(ctx, u, "")
	if err != nil {
		return "", err
	}
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// shasha returns the sha256sum of the SHA256SUMS file.
func shasha(ctx context.Context, version string) (string, error) {
	tmpl, err := template.New("").Parse(SHA256SUMSDownloadURLTemplate)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	args := map[string]string{
		"Version": version,
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
	/*
	 * go install example.com/cmd/foo@vX.Y.Z: bi.Main.Version="vX.Y.Z",                               vcs.revision is unset
	 * go install example.com/cmd/foo@latest: bi.Main.Version="vX.Y.Z",                               vcs.revision is unset
	 * go install example.com/cmd/foo@master: bi.Main.Version="vX.Y.Z-N.yyyyMMddhhmmss-gggggggggggg", vcs.revision is unset
	 * go install ./cmd/foo:                  bi.Main.Version="(devel)", vcs.revision="gggggggggggggggggggggggggggggggggggggggg"
	 *                                        vcs.time="yyyy-MM-ddThh:mm:ssZ", vcs.modified=("false"|"true")
	 */
	if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
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
