[[⬇️ **Download]**](https://github.com/reproducible-containers/repro-get/releases)
[[📖 **Quick start]**](#quick-start)
[[❓**FAQs & Troubleshooting]**](#faqs)

# `repro-get`: reproducible `apt`, `dnf`, `apk`, and `pacman`, with content-addressing

✅ HTTP and HTTPS

✅ Filesystems

✅ OCI (Open Container Initiative) registries

✅ IPFS

`repro-get` installs a specific snapshot of packages using `SHA256SUMS`, for the sake of [reproducible builds](https://reproducible-builds.org/):
```console
$ cat SHA256SUMS-amd64
35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc  pool/main/h/hello/hello_2.10-2_amd64.deb

$ repro-get install SHA256SUMS-amd64
(001/001) hello_2.10-2_amd64.deb Downloading from http://debian.notset.fr/snapshot/by-hash/SHA256/35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc
...
Preparing to unpack .../35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc ...
Unpacking hello (2.10-2) ...
Setting up hello (2.10-2) ...
```

`repro-get` supports the following distros:

| Distro                  | "Batteries included" | Support generating Dockerfiles | Support verifying package signatures |
| ----------------------- | -------------------- | ------------------------------ | ------------------------------------ |
| `debian`                | ✅                   | ✅                             | [❌](https://github.com/reproducible-containers/repro-get/issues/10) |
| `ubuntu`                | ❌                   | ❌                             | ❌                                   |
| `fedora` (Experimental) | ✅                   | ❌                             | ✅                                   |
| `alpine` (Experimental) | ❌                   | ❌                             | ✅                                   |
| `arch`                  | ✅                   | ✅                             | ✅                                   |

<details>
<summary> "Batteries included" for Debian, Fedora, and Arch Linux.</summary>

<p>

On Debian, the packages are fetched from the following URLs by default:
- `http://deb.debian.org/debian/{{.Name}}` for recent packages (fast, multi-arch, but ephemeral)
- `http://snapshot-cloudflare.debian.org/archive/debian/{{timeToDebianSnapshot .Epoch}}/{{.Name}}` for archived packages (slow, but persistent)

On Fedora: `https://kojipkgs.fedoraproject.org/packages/{{.Name}}` (multi-arch and persistent)

On Arch Linux: `https://archive.archlinux.org/packages/{{.Name}}` (multi-arch and persistent)

</p>

</details>

On other distros, the file provider has to be manually specified in the `--provider=...` flag for long-term persistence.

The following file providers are supported:
- HTTP/HTTPS URLs, such as `http://debian.notset.fr/snapshot/by-hash/SHA256/{{.SHA256}}`
- Filesystems, such as `file:///mnt/nfs/files/{{.Basename}}`, or `file:///mnt/nfs/blobs/{{.SHA256}}`
- [OCI-compliant container registries](#container-registries), such as `oci://ghcr.io/USERNAME/REPO`
- [IPFS](#ipfs) gateways, such as `http://ipfs.io/ipfs/{{.CID}}`

- - -
<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Quick start](#quick-start)
  - [Set up](#set-up)
  - [Installing packages with the hash file](#installing-packages-with-the-hash-file)
  - [Generating the hash file](#generating-the-hash-file)
  - [Updating the hash file](#updating-the-hash-file)
- [Advanced usage](#advanced-usage)
  - [Dockerfile](#dockerfile)
  - [Cache management](#cache-management)
    - [Populate](#populate)
    - [Export](#export)
    - [Import](#import)
    - [Clean](#clean)
  - [Container registries](#container-registries)
    - [Push](#push)
    - [Pull](#pull)
  - [IPFS](#ipfs)
    - [Push](#push-1)
    - [Pull](#pull-1)
- [FAQs](#faqs)
  - [Why do we need reproducibility?](#why-do-we-need-reproducibility)
  - [Why not just use `snapshot.debian.org` with `apt-get`?](#why-not-just-use-snapshotdebianorg-with-apt-get)
  - [Are container images "bit-to-bit" reproducible?](#are-container-images-bit-to-bit-reproducible)
  - [Does this work with Ubuntu?](#does-this-work-with-ubuntu)
  - [How to use HTTPS on Debian/Ubuntu?](#how-to-use-https-on-debianubuntu)
  - [Why not use HTTPS by default on Debian/Ubuntu?](#why-not-use-https-by-default-on-debianubuntu)
- [Acknowledgement](#acknowledgement)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->


## Quick start

### Set up
Download the latest binary release from https://github.com/reproducible-containers/repro-get/releases .

To install `repro-get` from source, install [Go](https://go.dev/dl/), run `make`, and `sudo make install`.
The recommended version of Go is written in the [`go.mod`](./go.mod) file.

The binary release can be reproduced locally by checking out the related tag and running `make artifacts.docker`.

### Installing packages with the hash file
Create the `SHA256SUMS-amd64` file for the [`hello`](https://packages.debian.org/bullseye/amd64/hello/download) package,
using the information from `apt-cache show hello`:
```
35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc  pool/main/h/hello/hello_2.10-2_amd64.deb
```

Then run `repro-get install SHA256SUMS-amd64`:
```console
$ repro-get install SHA256SUMS-amd64
(001/001) hello_2.10-2_amd64.deb Downloading from http://debian.notset.fr/snapshot/by-hash/SHA256/35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc
...
Preparing to unpack .../35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc ...
Unpacking hello (2.10-2) ...
Setting up hello (2.10-2) ...
```

See also [Dockerfile](#dockerfile) for running `repro-get` inside containers.

### Generating the hash file
> **Note**
>
> Make sure to run `apt-get update` before running `repro-get hash generate`.
>
> See also [Dockerfile](#dockerfile) for how to run `apt-get update` in a container image such as `debian:bullseye-yyyyMMdd`.

To generate the hash for all the installed packages, including the system packages:
```bash
repro-get hash generate >SHA256SUMS-amd64
```

To generate the hash for specific packages:
```bash
repro-get hash generate hello >SHA256SUMS-amd64
```

To generate the hash for newly installed packages:
```bash
repro-get hash generate >SHA256SUMS-amd64.old
apt-get install -y hello
repro-get hash generate --dedupe=SHA256SUMS-amd64.old >SHA256SUMS-amd64
```

### Updating the hash file
> **Note**
>
> Make sure to run `apt-get update` before running `repro-get hash update`.

To update the hash file:
```bash
repro-get hash update SHA256SUMS-amd64
```

## Advanced usage

### Dockerfile
> **Warning**
>
> `repro-get dockerfile generate` is an experimental feature.

The following example produces an image with `gcc`, using the packages from 2021-12-20.
```bash
# Generate "Dockerfile.generate-hash" and "Dockerfile" in the current directory
repro-get --distro=debian dockerfile generate . debian:bullseye-20211220 gcc build-essential

 Enable BuildKit
export DOCKER_BUILDKIT=1

# Generate "SHA256SUMS-amd64" file in the current directory (needed by the next step)
docker build --output . -f Dockerfile.generate-hash .

# Build the image
docker build .
```

See [`./examples/gcc`](./examples/gcc) for an example output.

See also [FAQs](#faqs) for "bit-to-bit" reproducibility of container images.

### Cache management
The cache directory (`--cache`) defaults to `/var/cache/repro-get`.

#### Populate
To populate the package files into the cache without installing them:
```bash
repro-get download SHA256SUMS-amd64
```

#### Export
To export the cached package files to the current directory:
```bash
repro-get cache export .
```

#### Import
To import package files in the current directory into the cache:
```bash
repro-get cache import .
```

#### Clean
To clean the cache:
```bash
repro-get cache clean
```

### Container registries

`repro-get` supports downloading package files from [OCI](https://github.com/opencontainers/distribution-spec)-compliant container registries.

> **Note**
>
> Make sure to create a container registry credential as `~/.docker/config.json` .
>
> - [GitHub Container Registry (`ghcr.io`)](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
> - [Others](https://github.com/containerd/nerdctl/blob/master/docs/registry.md#using-managed-registry-services)

#### Push
To push the package files into a container registry such as https://ghcr.io/ , use [ORAS](https://oras.land/cli/):
```bash
repro-get cache export .
oras push ghcr.io/USERNAME/dpkgs:latest *.deb
```

#### Pull
To pull and install packages from the registry:
```bash
repro-get --provider=oci://ghcr.io/USERNAME/dpkgs install SHA256SUMS-amd64
```

Tips about the `oci://...` provider strings:
- The provider string does not need contain the `:<TAG>@<DIGEST>` value, as `repro-get` ignores the container manifests.
- Defaults to HTTPS for non-localhost registries. Use `oci+http://...` scheme to disable HTTPS.

### IPFS

`repro-get` also supports uploading package files to IPFS, and downloading them from IPFS via an IPFS gateway such as `http://ipfs.io/ipfs/{{.CID}}` .

> **Note**
>
> The `ipfs` command ([Kubo](https://github.com/ipfs/kubo)) needs to be installed for pushing (not for pulling).

#### Push

Run `repro-get ipfs push` to push the package files, and update the hash file to include the IPFS CIDs:
```console
$ cat SHA256SUMS-amd64
35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc  pool/main/h/hello/hello_2.10-2_amd64.deb

$ repro-get ipfs push SHA256SUMS-amd64
35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc  /ipfs/QmRY19HEWeTJtRC6vAdz7rDfX3PjSMgXmd1KYi9guAACUj

$ cat SHA256SUMS-amd64
35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc  pool/main/h/hello/hello_2.10-2_amd64.deb
35b1508eeee9c1dfba798c4c04304ef0f266990f936a51f165571edf53325cbc  /ipfs/QmRY19HEWeTJtRC6vAdz7rDfX3PjSMgXmd1KYi9guAACUj
```

#### Pull
To pull and install packages from IPFS:
```bash
repro-get --provider=http://ipfs.io/ipfs/{{.CID}} install SHA256SUMS-amd64
```

The hash file must contain the `...  /ipfs/...` lines.

The hash file may contain multiple CIDs for a single SHA256, but only a single CID is used for pulling.

## FAQs
### Why do we need reproducibility?
For supply chain security.

If a binary can be bit-to-bit reproducible by multiple independent people, the binary (and its distributor) can be considered more trustable than others.

Achieving bit-to-bit reproducibility is still challenging (see below), but even "quasi-"reproducibility is useful for avoiding regressions that could be introduced by installing unexpected updates.

See also https://reproducible-builds.org/docs/buy-in/ .

### Why not just use `snapshot.debian.org` with `apt-get`?
Although it is already possible to reproduce a specific snapshot of Debian by specifying [`deb [...] http://snapshot.debian.org/archive/debian/yyyyMMddTHHmmssZ/ ... ...`](https://snapshot.debian.org/)
in `/etc/apt/sources.list`, this will cause a huge traffic on `snapshot.debian.org` when everybody begins to make builds reproducible.

`repro-get` mitigates this issue by content-addressing: A package file can be fetched from anywhere, such as HTTP(S) sites, local filesystems, OCI registries, or even IPFS, by its SHA256 (or CID) checksum.
Also, as the package files are verified by checksums, existing package files are not affected by potential GPG key leakage.

### Are container images "bit-to-bit" reproducible?
Yes, with BuildKit v0.11 or later.

See [`./hack/test-dockerfile-repro.sh`](./hack/test-dockerfile-repro.sh) for testing reproducibility.

However, it should be noted that the reproducibility is not guaranteed across different versions of BuildKit.
The host operating system version, filesystem configuration, etc. may affect reproducibility too.

### Does this work with Ubuntu?
Yes, but Ubuntu lacks an equivalent of http://snapshot.notset.fr/ , so you have to upload your cache to somewhere by yourself.

### How to use HTTPS on Debian/Ubuntu?
```bash
repro-get --provider='https://deb.debian.org/debian/{{.Name}},https://debian.notset.fr/snapshot/by-hash/SHA256/{{.SHA256}}' install
```

Using HTTPS needs the `ca-certificates` package to be installed.
The `ca-certificates` package is not installed by default in the [`debian`](https://hub.docker.com/_/debian) and [`ubuntu`](https://hub.docker.com/_/ubuntu)) images on Docker Hub.

### Why not use HTTPS by default on Debian/Ubuntu?
Because `apt-get` does not use HTTPS by default, either.
See [an archive of `whydoesaptnotusehttps.com`](https://web.archive.org/web/20200806030606/https://whydoesaptnotusehttps.com/) for the reason.

## Acknowledgement
A huge thanks to Frédéric Pierret ([@fepitre](https://github.com/fepitre)) for maintaining the [snapshot](https://github.com/fepitre/debian-snapshot) server http://snapshot.notset.fr/ .
Also huge thanks to maintainers of http://snapshot.debian.org/ , https://kojipkgs.fedoraproject.org/ , and other package snapshot servers.
`repro-get` could not be implemented without these snapshot servers.
