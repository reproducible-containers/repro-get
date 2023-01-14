package urlopener

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	refdocker "github.com/containerd/containerd/reference/docker"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/containerd/nerdctl/pkg/imgutil/dockerconfigresolver"
	"github.com/opencontainers/go-digest"
)

func New() *URLOpener {
	o := &URLOpener{
		resolvers: make(map[string]remotes.Resolver),
	}
	return o
}

type URLOpener struct {
	mu        sync.Mutex
	resolvers map[string]remotes.Resolver
}

var Schemes = []string{
	"http",
	"https",
	"file",
	"oci",
	"oci+http",
	"oci+https",
}

// Open opens the URL.
// The sha256sum argument is only used for resolving the OCI URLs.
// It is up to the caller to validate the sha256sum of the returned stream.
func (o *URLOpener) Open(ctx context.Context, u *url.URL, sha256sum string) (io.ReadCloser, int64, error) {
	switch u.Scheme {
	case "http", "https":
		req, err := http.NewRequest(http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, 0, err
		}
		req = req.WithContext(ctx)
		client := http.DefaultClient
		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, 0, fmt.Errorf("expected HTTP status %d for %q, got %s", http.StatusOK, u.Redacted(), resp.Status)
		}
		return resp.Body, resp.ContentLength, nil
	case "file":
		if u.User != nil || u.Host != "" || u.RawQuery != "" || u.Fragment != "" {
			return nil, 0, fmt.Errorf("invalid URL %q", u.Redacted())
		}
		file := u.Path
		st, err := os.Stat(file)
		if err != nil {
			return nil, 0, err
		}
		f, err := os.Open(file)
		return f, st.Size(), err
	case "oci", "oci+https", "oci+http":
		if sha256sum == "" {
			return nil, 0, errors.New("sha256sum must be provided as an argument of *URLOpener.Open()")
		}
		rawRef := strings.TrimPrefix(u.String(), u.Scheme+"://")
		ref, err := refdocker.ParseDockerRef(rawRef)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to parse OCI ref %q: %w", rawRef, err)
		}
		dgst := digest.NewDigestFromHex(digest.SHA256.String(), sha256sum)
		resolver, err := o.getOCIResolver(ctx, u.Scheme, ref)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get resolver for %q", u.Redacted())
		}
		// No need to call resolver.Resolve() here, as we do not care about the OCI manifests
		fetcher, err := resolver.Fetcher(ctx, ref.String())
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get fetcher for %v: %v: %w", dgst, ref, err)
		}
		r, desc, err := fetcher.(remotes.FetcherByDigest).FetchByDigest(ctx, dgst)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get reader for %v: %v: %w", dgst, ref, err)
		}
		return r, desc.Size, nil
	default:
		return nil, 0, fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
}

func (o *URLOpener) getOCIResolver(ctx context.Context, scheme string, ref refdocker.Named) (remotes.Resolver, error) {
	refDomain := refdocker.Domain(ref)
	o.mu.Lock()
	k := scheme + "://" + refDomain
	resolver, ok := o.resolvers[k]
	o.mu.Unlock()
	if ok {
		return resolver, nil
	}
	var dOpts []dockerconfigresolver.Opt
	switch scheme {
	case "oci+http":
		dOpts = append(dOpts, dockerconfigresolver.WithPlainHTTP(true))
	case "oci+https":
		if ok, _ := docker.MatchLocalhost(refDomain); ok {
			// https://github.com/containerd/nerdctl/blob/v0.23.0/pkg/imgutil/dockerconfigresolver/dockerconfigresolver.go#L130-L137
			return nil, fmt.Errorf("https is not supported for localhost %q (FIXME)", refDomain)
		}
	case "oci":
		// NOP
	default:
		return nil, fmt.Errorf("expected oci://, oci+http://, or oci+https, got %q", scheme)
	}
	var err error
	resolver, err = dockerconfigresolver.New(ctx, refDomain, dOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create a resolver for refDomain=%q (ref=%q): %w", refDomain, ref, err)
	}
	o.mu.Lock()
	o.resolvers[k] = resolver
	o.mu.Unlock()
	return resolver, nil
}
