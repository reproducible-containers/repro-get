package cache

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
	"gotest.tools/v3/assert"
)

type testBlob struct {
	basename string
	b        []byte
	sha256   string
}

func newTestBlob(basename string) *testBlob {
	b := []byte("blob-" + basename)
	sha256 := digest.SHA256.FromBytes(b).Encoded()
	return &testBlob{
		basename: basename,
		b:        b,
		sha256:   sha256,
	}
}

func newTestBlobs(basenames ...string) (bySHA256 map[string]*testBlob) {
	bySHA256 = make(map[string]*testBlob, len(basenames))
	for _, basename := range basenames {
		basename := basename
		testBlob := newTestBlob(basename)
		bySHA256[testBlob.sha256] = testBlob
	}
	return
}

type httptestServer struct {
	t            testing.TB
	ociImageName string
	*httptest.Server
}

func newTestHTTPServer(t testing.TB, blobsBySHA256 map[string]*testBlob) *httptestServer {
	const ociImageName = "dummy-image"
	return &httptestServer{
		t:            t,
		ociImageName: ociImageName,
		Server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			badReq := func(err error) {
				if err != nil {
					t.Error(err)
				}
				w.WriteHeader(http.StatusBadRequest)
			}
			reply := func(blob *testBlob) {
				if _, err := io.Copy(w, bytes.NewReader(blob.b)); err != nil {
					t.Error(err)
				}
			}
			matchedByName, err := path.Match("/basenames/*", r.URL.Path)
			if err != nil {
				badReq(err)
				return
			}
			if matchedByName {
				for _, blob := range blobsBySHA256 {
					if blob.basename == path.Base(r.URL.Path) {
						reply(blob)
						return
					}
				}
				w.WriteHeader(http.StatusNotFound)
				return
			}
			matchedBySHA256, err := path.Match("/blobs/sha256/*", r.URL.Path)
			if err != nil {
				badReq(err)
				return
			}
			matchedOCI, err := path.Match("/v2/"+ociImageName+"/blobs/sha256:*", r.URL.Path)
			if err != nil {
				badReq(err)
				return
			}
			if !matchedBySHA256 && !matchedOCI {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			sha256 := strings.TrimPrefix(path.Base(r.URL.Path), "sha256:")
			blob, ok := blobsBySHA256[sha256]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			reply(blob)
		})),
	}
}

func (srv *httptestServer) basenameURL(blob *testBlob) *url.URL {
	srv.t.Helper()
	rawURL := fmt.Sprintf("%s/basenames/%s", srv.URL, blob.basename)
	u, err := url.Parse(rawURL)
	assert.NilError(srv.t, err)
	return u
}

func (srv *httptestServer) digestURL(blob *testBlob) *url.URL {
	srv.t.Helper()
	rawURL := fmt.Sprintf("%s/blobs/sha256/%s", srv.URL, blob.sha256)
	u, err := url.Parse(rawURL)
	assert.NilError(srv.t, err)
	return u
}

func (srv *httptestServer) ociURL() *url.URL {
	srv.t.Helper()
	rawURL := strings.Replace(srv.URL, "http://", "oci://", 1) + "/" + srv.ociImageName
	u, err := url.Parse(rawURL)
	assert.NilError(srv.t, err)
	return u
}

func testCacheEnsure(t testing.TB, blobsBySHA256 map[string]*testBlob, newBlobURL func(*testBlob) *url.URL) {
	t.Helper()
	cacheDir := t.TempDir()
	cache, err := New(cacheDir)
	assert.NilError(t, err)
	assert.Equal(t, cacheDir, cache.Dir())

	ctx := context.TODO()
	for _, blob := range blobsBySHA256 {
		u := newBlobURL(blob)
		for i := 0; i < 2; i++ { // run twice to test idempotency
			assert.NilError(t, cache.Ensure(ctx, u, blob.sha256))
			ok, err := cache.Cached(blob.sha256)
			assert.NilError(t, err)
			assert.Equal(t, true, ok)
		}
		assert.Check(t, cache.Ensure(ctx, u, digest.SHA256.FromString("wrong").Encoded()) != nil)
	}

	testCacheDir(t, cache, blobsBySHA256)
}

func testCacheDir(t testing.TB, cache *Cache, blobsBySHA256 map[string]*testBlob) {
	t.Helper()
	for _, blob := range blobsBySHA256 {
		blobRel, err := cache.BlobRelPath(blob.sha256)
		assert.NilError(t, err)
		assert.Equal(t, "blobs/sha256/"+blob.sha256, blobRel)
		b, err := os.ReadFile(filepath.Join(cache.Dir(), blobRel))
		assert.NilError(t, err)
		assert.DeepEqual(t, blob.b, b)
	}
}

func TestCacheEnsure(t *testing.T) {
	blobsBySHA256 := newTestBlobs("foo", "bar", "baz")
	testServer := newTestHTTPServer(t, blobsBySHA256)
	defer testServer.Close()

	t.Run("http", func(t *testing.T) {
		testCacheEnsure(t, blobsBySHA256, testServer.digestURL)
		testCacheEnsure(t, blobsBySHA256, testServer.basenameURL)
	})

	t.Run("oci", func(t *testing.T) {
		u := testServer.ociURL()
		testCacheEnsure(t, blobsBySHA256, func(*testBlob) *url.URL {
			// u does not contain manifest digest
			return u
		})
	})

	t.Run("file", func(t *testing.T) {
		testDir := t.TempDir()
		for _, blob := range blobsBySHA256 {
			f := filepath.Join(testDir, blob.sha256)
			assert.NilError(t, os.WriteFile(f, blob.b, 0644))
		}
		testCacheEnsure(t, blobsBySHA256, func(blob *testBlob) *url.URL {
			rawURL := "file://" + filepath.Join(testDir, blob.sha256)
			u, err := url.Parse(rawURL)
			assert.NilError(t, err)
			return u
		})
	})
}

func TestCacheExportImport(t *testing.T) {
	ctx := context.TODO()
	cacheDir := t.TempDir()
	cache, err := New(cacheDir)
	assert.NilError(t, err)
	basenames := []string{"foo", "bar", "baz"}
	sort.Strings(basenames)
	blobsBySHA256 := newTestBlobs(basenames...)
	testServer := newTestHTTPServer(t, blobsBySHA256)
	defer testServer.Close()
	for _, blob := range blobsBySHA256 {
		u := testServer.basenameURL(blob)
		assert.NilError(t, cache.Ensure(ctx, u, blob.sha256))
	}

	mapByBasename := make(map[string]string)
	for _, f := range blobsBySHA256 {
		mapByBasename[f.basename] = f.sha256
	}

	exportDir := t.TempDir()
	exported, err := cache.Export(exportDir)
	assert.NilError(t, err)
	assert.DeepEqual(t, mapByBasename, exported)

	for basename := range exported {
		f := filepath.Join(exportDir, basename)
		b, err := os.ReadFile(f)
		assert.NilError(t, err)
		assert.DeepEqual(t, blobsBySHA256[mapByBasename[basename]].b, b)
	}

	t.Run("ImportByDir", func(t *testing.T) {
		cache2Dir := t.TempDir()
		cache2, err := New(cache2Dir)
		assert.NilError(t, err)

		for i := 0; i < 2; i++ { // run twice to test idempotency
			imported, err := cache2.Import(exportDir)
			assert.NilError(t, err)
			assert.DeepEqual(t, mapByBasename, imported)
			testCacheDir(t, cache2, blobsBySHA256)
		}
	})

	t.Run("ImportByFile", func(t *testing.T) {
		cache2Dir := t.TempDir()
		cache2, err := New(cache2Dir)
		assert.NilError(t, err)
		for i := 0; i < 2; i++ { // run twice to test idempotency
			for basename, sha256sum := range exported {
				imported, err := cache2.Import(filepath.Join(exportDir, basename))
				assert.NilError(t, err)
				assert.DeepEqual(t, map[string]string{basename: sha256sum}, imported)
			}
			testCacheDir(t, cache2, blobsBySHA256)
		}
	})
}
