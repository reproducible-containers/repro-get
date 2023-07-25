package filespec

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	pkgepoch "github.com/containerd/containerd/pkg/epoch"
	"github.com/opencontainers/go-digest"
	"github.com/reproducible-containers/repro-get/pkg/apkutil"
	"github.com/reproducible-containers/repro-get/pkg/dpkgutil"
	"github.com/reproducible-containers/repro-get/pkg/pacmanutil"
	"github.com/reproducible-containers/repro-get/pkg/rpmutil"
	"github.com/reproducible-containers/repro-get/pkg/sha256sums"
	"github.com/sirupsen/logrus"
)

func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("file name is empty")
	}
	if path.IsAbs(name) {
		return fmt.Errorf("file name %q must not be absolute", name)
	}
	if path.Clean(name) != name {
		return fmt.Errorf("file name %q must be clean", name)
	}
	base := path.Base(name)
	if strings.HasPrefix(base, ".") {
		// Dot names are reserved for future extension
		return fmt.Errorf("file name %q must not start with \".\"", base)
	}
	return nil
}

type opts struct {
	cid   string
	epoch *time.Time
}

type Option func(o *opts)

func WithCID(cid string) Option {
	return func(o *opts) {
		o.cid = cid
	}
}

func WithEpoch(epoch *time.Time) Option {
	return func(o *opts) {
		o.epoch = epoch
	}
}

func New(name, sha256 string, options ...Option) (*FileSpec, error) {
	var opts opts
	for _, o := range options {
		o(&opts)
	}
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	if err := digest.SHA256.Validate(sha256); err != nil {
		return nil, err
	}
	sp := &FileSpec{
		Name:     name,
		Basename: filepath.Base(name),
		SHA256:   sha256,
		CID:      opts.cid,
		Epoch:    opts.epoch,
	}
	switch {
	case strings.HasSuffix(name, ".deb"):
		dpkg, err := dpkgutil.ParseFilename(name)
		if err != nil {
			return sp, err
		}
		sp.Dpkg = dpkg
	case strings.HasSuffix(name, ".rpm"):
		rpm, err := rpmutil.ParseFilename(name)
		if err != nil {
			return sp, err
		}
		sp.RPM = rpm
	case strings.HasSuffix(name, ".apk"):
		apk, err := apkutil.ParseFilename(name)
		if err != nil {
			return sp, err
		}
		sp.APK = apk
	case strings.HasSuffix(name, ".pkg.tar.zst"):
		pacman, err := pacmanutil.ParseFilename(name)
		if err != nil {
			return sp, err
		}
		sp.Pacman = pacman
	}
	return sp, nil
}

type FileSpec struct {
	Name     string             `json:"Name"`            // "pool/main/h/hello/hello_2.10-2_amd64.deb"
	Basename string             `json:"Basename"`        // "hello_2.10-2_amd64.deb"
	SHA256   string             `json:"SHA256"`          // "35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc"
	CID      string             `json:"CID,omitempty"`   // IPFS CID
	Epoch    *time.Time         `json:"Epoch,omitempty"` // Timestamp of SHA256SUMS, or $SOURCE_DATE_EPOCH
	Dpkg     *dpkgutil.Dpkg     `json:"Dpkg,omitempty"`
	RPM      *rpmutil.RPM       `json:"RPM,omitempty"`
	APK      *apkutil.APK       `json:"APK,omitempty"`
	Pacman   *pacmanutil.Pacman `json:"Pacman,omitempty"`
}

var FileSpecTemplateFuncMap = template.FuncMap{
	"timeToInt": func(tm *time.Time) (string, error) {
		if tm == nil {
			return "", errors.New("nil time")
		}
		return fmt.Sprintf("%d", tm.UTC().Unix()), nil
	},
	"timeToDebianSnapshot": func(tm *time.Time) (string, error) {
		if tm == nil {
			return "", errors.New("nil time")
		}
		u := tm.UTC()
		return fmt.Sprintf("%04d%02d%02dT%02d%02d%02dZ",
			u.Year(), u.Month(), u.Day(), u.Hour(), u.Minute(), u.Second()), nil
	},
}

func (sp FileSpec) URL(provider string) (*url.URL, error) {

	// FIXME: find a more robust way to error out when a template property is empty
	if strings.Contains(provider, ".CID") && sp.CID == "" {
		return nil, fmt.Errorf("no CID is known for sha256 %q", sp.SHA256)
	}

	tmpl, err := template.New("").Funcs(FileSpecTemplateFuncMap).Parse(provider)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	if err = tmpl.Execute(&b, sp); err != nil {
		return nil, err
	}
	s := b.String()

	u, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q as a URL: %w", s, err)
	}

	isOCI := u.Scheme == "oci" || strings.HasPrefix(u.Scheme, "oci+")

	if isOCI {
		if strings.Contains(s, "@sha256:") {
			logrus.Warnf("No need to provide the '@sha256...' suffix in an OCI provider string, got %q", s)
		}
	} else {
		if s == provider {
			return nil, fmt.Errorf("invalid provider %q", provider)
		}
	}
	return u, nil
}

// PseudoFilename is prefixed with "/ipfs/".
// e.g., "/ipfs/QmbFMke1KXqnYyBBWxB74N4c5SBnJMVAiMNRcGu6x1AwQH".
type PseudoFilename struct {
	CID string
}

// ParsePseudoFilename parses a pseudo file name.
func ParsePseudoFilename(s string) *PseudoFilename {
	if !strings.HasPrefix(s, "/ipfs/") {
		return nil
	}
	sp := strings.Split(s, "/")
	if len(sp) != 3 {
		logrus.Warnf("Invalid pseudo IPFS filename: expected \"/ipfs/<CID>\", got %q", s)
		return nil
	}
	return &PseudoFilename{
		CID: sp[2],
	}
}

type hashMapOpts struct {
	epoch *time.Time
}

type HashMapOption func(o *hashMapOpts)

func WithHashMapEpoch(epoch *time.Time) HashMapOption {
	return func(o *hashMapOpts) {
		o.epoch = epoch
	}
}

// NewFromSHA256SUMS returns a file spec map from the sha256sums map.
// The key of the returned map is a file name such as "pool/main/h/hello/hello_2.10-2_amd64.deb"".
// The key does not contain "pseudo" file names prefixed with "/ipfs/".
func NewFromSHA256SUMS(sha256sumsMapByFilename map[string]string, options ...HashMapOption) (map[string]*FileSpec, error) {
	var opts hashMapOpts
	for _, o := range options {
		o(&opts)
	}

	var allFilenames []string // contains "pseudo" file names too
	for f := range sha256sumsMapByFilename {
		allFilenames = append(allFilenames, f)
	}
	sort.Strings(allFilenames)
	entries := make(map[string]*FileSpec)
	cids := make(map[string]string) // key: sha256, value: cid
	for _, filenameMaybePseudo := range allFilenames {
		sum := sha256sumsMapByFilename[filenameMaybePseudo]
		if pseudo := ParsePseudoFilename(filenameMaybePseudo); pseudo != nil {
			if oldCID := cids[sum]; oldCID != "" {
				logrus.Warnf("Multiple CIDs found for SHA256 %q, discarding CID %q, using %q", sum, oldCID, pseudo.CID)
			}
			cids[sum] = pseudo.CID
			continue
		}
		filename := filenameMaybePseudo
		cid := cids[sum] // often empty
		sp, err := New(filename, sum, WithCID(cid), WithEpoch(opts.epoch))
		if err != nil {
			return nil, err
		}
		entries[filename] = sp
	}
	return entries, nil
}

func NewFromSHA256SUMSFiles(fnames ...string) (map[string]*FileSpec, error) {
	var sourceDateEpoch *time.Time
	if v, err := pkgepoch.SourceDateEpoch(); err == nil {
		sourceDateEpoch = v
	}
	res := make(map[string]*FileSpec)
	for _, fname := range fnames {
		subRes, err := newFromSHA256SUMSFile(fname, sourceDateEpoch)
		if err != nil {
			return res, fmt.Errorf("failed to parse %q: %w", fname, err)
		}
		for k, v := range subRes {
			res[k] = v
		}
	}
	return res, nil
}

func newFromSHA256SUMSFile(fname string, sourceDateEpoch *time.Time) (map[string]*FileSpec, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sums, err := sha256sums.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q as SHA256SUMS: %w", fname, err)
	}
	st, err := f.Stat() // follow symlinks
	if err != nil {
		return nil, err
	}
	epoch := st.ModTime().UTC()
	if sourceDateEpoch != nil {
		epoch = *sourceDateEpoch
	}
	return NewFromSHA256SUMS(sums, WithHashMapEpoch(&epoch))
}
