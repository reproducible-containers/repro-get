# Example: hello

The `SHA256SUMS-*` files in this directory corresponds to https://packages.debian.org/bullseye/hello .

The `Dockerfile` was generated and verified with the following commands:
```bash
# Generate "Dockerfile" in the current directory
repro-get --distro=debian dockerfile generate . debian:bullseye-20211220

# Copy the repro-get binary into the current directory
cp $(command -v repro-get) ./repro-get.linux-amd64

# Enable BuildKit
export DOCKER_BUILDKIT=1

# Build the image
docker build .

# Clean up
rm -f ./repro-get.linux-amd64
```
