// Package cache provides the blob cache.
//
// File name convention:
//
//   - blobs/sha256/*.tmp:    tmp files
//
//   - blobs/sha256/<SHA256>: verified blobs
//
//   - metadata/sha256/<SHA256> : metadata of the blob (optional)
//
//   - digests/by-url-sha256/<SHA256-OF-URL> : digest of the blob (optional, note that URL is not always unique)
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/containerd/continuity/fs"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/opencontainers/go-digest"
	"github.com/reproducible-containers/repro-get/pkg/progressbar"
	"github.com/reproducible-containers/repro-get/pkg/urlopener"
	"github.com/sirupsen/logrus"
)

type Metadata struct {
	Basename string
}

func ValidateMetadata(m *Metadata) error {
	if m == nil {
		return nil
	}
	if filepath.Base(m.Basename) != m.Basename {
		return fmt.Errorf("invalid basename: %q", m.Basename)
	}
	return nil
}

const (
	BlobsSHA256RelPath    = "blobs/sha256"
	MetadataSHA256RelPath = "metadata/sha256"
	ReverseURLRelPath     = "digests/by-url-sha256"
)

func New(dir string) (*Cache, error) {
	if os.PathSeparator != '/' {
		return nil, fmt.Errorf("expected os.PathSeparator to be '/', got %c", os.PathSeparator)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	for _, f := range []string{BlobsSHA256RelPath, MetadataSHA256RelPath, ReverseURLRelPath} {
		subDir := filepath.Join(dir, f) // no need to use securejoin (const)
		if err := os.MkdirAll(subDir, 0755); err != nil {
			return nil, err
		}
	}
	c := &Cache{
		dir:       dir,
		urlOpener: urlopener.New(),
	}
	return c, nil
}

type Cache struct {
	dir       string
	urlOpener *urlopener.URLOpener
}

func (c *Cache) Dir() string {
	return c.dir
}

// BlobRelPath returns a clean relative path like "blobs/sha256/<SHA256>".
// The caller should append this path to c.Dir().
// The returned path may not exist.
// If it exists, its digest must have been already verified.
func (c *Cache) BlobRelPath(sha256sum string) (string, error) {
	if err := digest.SHA256.Validate(sha256sum); err != nil {
		return "", err
	}
	return securejoin.SecureJoin(BlobsSHA256RelPath, sha256sum)
}

func (c *Cache) BlobAbsPath(sha256sum string) (string, error) {
	rel, err := c.BlobRelPath(sha256sum)
	if err != nil {
		return "", err
	}
	return filepath.Join(c.dir, rel), nil // no need to use securejoin (rel is verified)
}

func (c *Cache) MetadataFileRelPath(sha256sum string) (string, error) {
	if err := digest.SHA256.Validate(sha256sum); err != nil {
		return "", err
	}
	return securejoin.SecureJoin(MetadataSHA256RelPath, sha256sum)
}

func (c *Cache) MetadataFileAbsPath(sha256sum string) (string, error) {
	rel, err := c.MetadataFileRelPath(sha256sum)
	if err != nil {
		return "", err
	}
	return filepath.Join(c.dir, rel), nil // no need to use securejoin (rel is verified)
}

func (c *Cache) ReverseURLFileRelPath(u *url.URL) (string, error) {
	// u.Redacted is used for consistency with the URL files
	sha256OfURL := digest.SHA256.FromBytes([]byte(u.Redacted())).Encoded()
	return securejoin.SecureJoin(ReverseURLRelPath, sha256OfURL)
}

func (c *Cache) ReverseURLFileAbsPath(u *url.URL) (string, error) {
	rel, err := c.ReverseURLFileRelPath(u)
	if err != nil {
		return "", err
	}
	return filepath.Join(c.dir, rel), nil // no need to use securejoin (rel is verified)
}

func (c *Cache) Cached(sha256sum string) (bool, error) {
	blob, err := c.BlobAbsPath(sha256sum)
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(blob); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *Cache) Ensure(ctx context.Context, u *url.URL, sha256sum string, m *Metadata) error {
	if err := ValidateMetadata(m); err != nil {
		return err
	}
	blob, err := c.BlobAbsPath(sha256sum) // also verifies sha256sum string representation
	if err != nil {
		return err
	}
	if _, err := os.Stat(blob); err == nil {
		// sha256sum is verified on the initial caching
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	tmpW, err := os.CreateTemp(filepath.Dir(blob), ".download-*.tmp")
	if err != nil {
		return err
	}
	defer func() {
		tmpW.Close()
		os.Remove(tmpW.Name())
	}()

	r, sz, err := c.urlOpener.Open(ctx, u, sha256sum)
	if err != nil {
		return fmt.Errorf("failed to open URL %q: %w", u.Redacted(), err)
	}
	defer r.Close()

	bar, err := progressbar.New(sz)
	if err != nil {
		return err
	}

	digester := digest.SHA256.Digester()
	hasher := digester.Hash()
	mw := io.MultiWriter(tmpW, hasher)

	bar.Start()
	if _, err = io.Copy(mw, bar.NewProxyReader(r)); err != nil {
		return fmt.Errorf("failed to copy %d bytes: %w", sz, err)
	}
	bar.Finish()

	actualSHA256SUM := digester.Digest().Encoded()
	if actualSHA256SUM != sha256sum {
		return fmt.Errorf("expected sha256sum %q, got %q", sha256sum, actualSHA256SUM)
	}

	if err = tmpW.Sync(); err != nil {
		return err
	}
	if err = tmpW.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmpW.Name(), blob); err != nil {
		return err
	}
	if err := c.writeMetadataFiles(sha256sum, u, m); err != nil {
		return err
	}
	return nil
}

func (c *Cache) Export(dir string) (map[string]string, error) {
	blobs, err := os.ReadDir(filepath.Join(c.dir, BlobsSHA256RelPath)) // no need to use securejoin (const)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	exported := make(map[string]string)
	for _, f := range blobs {
		if f.IsDir() {
			continue
		}
		sha256sum := f.Name()
		if strings.HasPrefix(sha256sum, ".") || strings.HasSuffix(sha256sum, ".tmp") {
			continue
		}
		if err = digest.SHA256.Validate(sha256sum); err != nil {
			logrus.WithError(err).Errorf("Invalid sha256sum %q", sha256sum)
			continue
		}
		cpSrc := filepath.Join(c.dir, BlobsSHA256RelPath, sha256sum) // no need to use securejoin (sha256sum is verified)
		basename := "UNKNOWN-" + sha256sum
		if m, err := c.MetadataBySHA256(sha256sum); err == nil && m.Basename != "" {
			basename = filepath.Base(m.Basename)
		} else {
			logrus.WithError(err).Warnf("Failed to get the original basename of %s", sha256sum)
		}
		cpDst, err := securejoin.SecureJoin(dir, basename)
		if err != nil {
			return exported, err
		}
		if _, err := os.Lstat(cpDst); !errors.Is(err, os.ErrNotExist) {
			logrus.Errorf("Avoiding to overwrite existing file %q", cpDst)
			continue
		}
		if err = fs.CopyFile(cpDst, cpSrc); err != nil {
			return exported, err
		}
		exported[basename] = sha256sum
	}
	return exported, nil
}

// Import imports local directories or files, and returns map[basename]sha256sum .
func (c *Cache) Import(dirOrFiles ...string) (map[string]string, error) {
	m := make(map[string]string)
	for _, f := range dirOrFiles {
		xM, err := c.import1(f)
		if err != nil {
			return m, nil
		}
		for k, v := range xM {
			if conflict, ok := m[k]; ok {
				return m, fmt.Errorf("conflict: basename=%q, sha256sum=%q vs %q", k, v, conflict)
			}
			m[k] = v
		}
	}
	return m, nil
}

func (c *Cache) import1(dirOrFile string) (map[string]string, error) {
	st, err := os.Stat(dirOrFile)
	if err != nil {
		return nil, err
	}
	if st.IsDir() {
		return c.importDir(dirOrFile)
	}
	file := dirOrFile
	sha256sum, err := c.importFile(file)
	if err != nil {
		return nil, err
	}
	return map[string]string{filepath.Base(file): sha256sum}, nil
}

func (c *Cache) importDir(dir string) (map[string]string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string)
	for _, ent := range ents {
		basename := ent.Name()
		nameFull, err := securejoin.SecureJoin(dir, basename)
		if err != nil {
			return m, err
		}
		if ent.IsDir() {
			mRec, err := c.importDir(nameFull)
			if err != nil {
				return m, err
			}
			for k, v := range mRec {
				if conflict, ok := m[k]; ok {
					return m, fmt.Errorf("conflict: basename=%q, sha256sum=%q vs %q", k, v, conflict)
				}
				m[k] = v
			}
		} else {
			sha256sum, err := c.importFile(nameFull)
			if err != nil {
				return m, err
			}
			m[basename] = sha256sum
		}
	}
	return m, nil
}

func (c *Cache) importFile(nameFull string) (sha256sum string, err error) {
	u, err := url.Parse("file://" + nameFull)
	if err != nil {
		return "", err
	}
	m := &Metadata{
		Basename: filepath.Base(nameFull),
	}
	return c.ImportWithURL(u, m)
}

// ImportWithReader imports from the reader.
// Does not create the metadata files.
func (c *Cache) ImportWithReader(r io.Reader) (sha256sum string, err error) {
	blobsSHA256Dir := filepath.Join(c.dir, BlobsSHA256RelPath) // no need to use securejoin (const)
	tmpW, err := os.CreateTemp(blobsSHA256Dir, ".import-*.tmp")
	if err != nil {
		return "", err
	}
	defer func() {
		tmpW.Close()
		os.Remove(tmpW.Name())
	}()
	digester := digest.SHA256.Digester()
	hasher := digester.Hash()
	mw := io.MultiWriter(tmpW, hasher)
	if _, err = io.Copy(mw, r); err != nil {
		return "", err
	}
	sha256sum = digester.Digest().Encoded()
	blob, err := c.BlobAbsPath(sha256sum)
	if err != nil {
		return "", err
	}
	if err = tmpW.Sync(); err != nil {
		return "", err
	}
	if err = tmpW.Close(); err != nil {
		return "", err
	}
	if err = os.Rename(tmpW.Name(), blob); err != nil {
		return "", err
	}
	return sha256sum, nil
}

func (c *Cache) ImportWithURL(u *url.URL, m *Metadata) (sha256sum string, err error) {
	if err := ValidateMetadata(m); err != nil {
		return "", err
	}
	r, _, err := c.urlOpener.Open(context.TODO(), u, "")
	if err != nil {
		return "", err
	}
	defer r.Close()
	sha256sum, err = c.ImportWithReader(r)
	if err != nil {
		return "", err
	}
	err = c.writeMetadataFiles(sha256sum, u, m)
	return sha256sum, err
}

// writeMetadataFiles writes metadata files and reverse URL files.
// Note that a URL is not unique.
// Existing files are overwritten.
func (c *Cache) writeMetadataFiles(sha256sum string, u *url.URL, m *Metadata) error {
	if m != nil {
		j, err := json.Marshal(m)
		if err != nil {
			return err
		}
		metadataFileAbs, err := c.MetadataFileAbsPath(sha256sum)
		if err != nil {
			return err
		}
		if err = os.WriteFile(metadataFileAbs, j, 0644); err != nil {
			return fmt.Errorf("failed to create %q: %w", metadataFileAbs, err)
		}
	}
	if u != nil {
		revURLFileAbs, err := c.ReverseURLFileAbsPath(u)
		if err != nil {
			return err
		}
		if err = os.WriteFile(revURLFileAbs, []byte("sha256:"+sha256sum), 0644); err != nil {
			return fmt.Errorf("failed to create %q: %w", revURLFileAbs, err)
		}
	}
	return nil
}

// MetadataBySHA256 returns the metadata for the blob.
// Not always available.
func (c *Cache) MetadataBySHA256(sha256sum string) (*Metadata, error) {
	metadataFileAbs, err := c.MetadataFileAbsPath(sha256sum)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(metadataFileAbs)
	if err != nil {
		return nil, err
	}
	var m Metadata
	if err = json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// SHA256ByOriginURL returns the sha256sum by the origin URL.
// Not always available.
// Do not use this unless you are sure that the URL is unique.
func (c *Cache) SHA256ByOriginURL(u *url.URL) (string, error) {
	revUrlFileAbs, err := c.ReverseURLFileAbsPath(u)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(revUrlFileAbs)
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(b))
	d, err := digest.Parse(s)
	if err != nil {
		return "", err
	}
	if d.Algorithm() != digest.SHA256 {
		return "", fmt.Errorf("expected algorithm %q, got %q (%q)", digest.SHA256, d.Algorithm(), d)
	}
	return d.Encoded(), nil
}
