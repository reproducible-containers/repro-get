package ocidistutil

import (
	"context"
	"fmt"

	refdocker "github.com/containerd/containerd/reference/docker"
	"github.com/containerd/nerdctl/pkg/imgutil/dockerconfigresolver"
)

// RefWithDigest appends "@sha256:.." to the ref.
func RefWithDigest(ctx context.Context, rawRef string) (string, error) {
	ref, err := refdocker.ParseDockerRef(rawRef)
	if err != nil {
		return "", fmt.Errorf("failed to parse OCI ref %q: %w", rawRef, err)
	}
	refDomain := refdocker.Domain(ref)
	resolver, err := dockerconfigresolver.New(ctx, refDomain)
	if err != nil {
		return "", fmt.Errorf("failed to create a resolver for refDomain=%q (ref=%q): %w", refDomain, ref, err)
	}
	_, desc, err := resolver.Resolve(ctx, ref.String())
	if err != nil {
		return "", err
	}
	resolvedWithDigest, err := refdocker.WithDigest(ref, desc.Digest)
	if err != nil {
		return "", err
	}
	return resolvedWithDigest.String(), nil
}
