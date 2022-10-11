# Example: gcc

The files in this directory were generated and verified with the following commands:
```bash
# Generate "Dockerfile.generate-hash" and "Dockerfile" in the current directory
repro-get --distro=debian dockerfile generate . debian:bullseye-20211220 $(cat PACKAGES)

# Enable BuildKit
export DOCKER_BUILDKIT=1

# Generate "SHA256SUMS-amd64" in the current directory
docker build --output . -f Dockerfile.generate-hash .

# Build the image
docker build .
```
