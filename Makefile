# Files are installed under $(DESTDIR)/$(PREFIX)
PREFIX ?= /usr/local
DEST := $(shell echo "$(DESTDIR)/$(PREFIX)" | sed 's:///*:/:g; s://*$$::')

VERSION ?=$(shell git describe --match 'v[0-9]*' --dirty='.m' --always --tags)
PACKAGE := github.com/reproducible-containers/repro-get

export CGO_ENABLED ?= 0
export DOCKER_BUILDKIT := 1
export SOURCE_DATE_EPOCH ?= $(shell git log -1 --pretty=%ct)

GO ?= go
GO_BUILD ?= $(GO) build -trimpath -ldflags="-s -w -X $(PACKAGE)/pkg/version.Version=$(VERSION)"
DOCKER ?= docker
DOCKER_BUILD ?= $(DOCKER) build
UPX ?= upx --best --lzma
TAR ?= tar
# FROM https://reproducible-builds.org/docs/archives/
TAR_C ?= $(TAR) -c --sort=name --owner=0 --group=0 --numeric-owner --pax-option=exthdr.name=%d/PaxHeaders/%f,delete=atime,delete=ctime --mtime=@$(SOURCE_DATE_EPOCH)

.PHONY: all
all: binaries

.PHONY: binaries
binaries: _output/bin/repro-get

.PHONY: _output/bin/repro-get
_output/bin/repro-get:
	$(GO_BUILD) -o $@ ./cmd/repro-get

.PHONY: install
install: uninstall
	mkdir -p "$(DEST)"
	install _output/bin/repro-get "$(DEST)/bin/repro-get"

.PHONY: uninstall
uninstall:
	rm -rf "$(DEST)/bin/repro-get"

.PHONY: clean
clean:
	rm -rf _output _artifacts

.PHONY: generate
generate:
	$(MAKE) -C examples

.PHONY: integration
integration:
	$(DOCKER_BUILD) -f Makefile.d/Dockerfile.integration .

.PHONY: artifacts
artifacts: clean artifacts-linux artifacts-misc
	(cd _artifacts ; sha256sum *) > SHA256SUMS
	mv SHA256SUMS _artifacts/SHA256SUMS
	touch -d @$(SOURCE_DATE_EPOCH) _artifacts/SHA256SUMS

.PHONY: artifacts-linux
artifacts-linux:
	@# Filename convention: "repro-get-${VERSION}.${TARGETOS}-${TARGETARCH}${TARGETVARIANT:+-${TARGETVARIANT}}" .
	@# TARGETOS, TARGETARCH, and TARETVARIANT are Dockerfile's built-in variables.
	mkdir -p _artifacts
	export GOOS=linux
	# Docker: "linux/amd64",   Debian: "amd64"
	GOARCH=amd64            $(GO_BUILD) -o _artifacts/repro-get-$(VERSION).linux-amd64   ./cmd/repro-get
	# Docker: "linux/arm/v7",  Debian: "armhf"
	GOARCH=arm      GOARM=7 $(GO_BUILD) -o _artifacts/repro-get-$(VERSION).linux-arm-v7  ./cmd/repro-get
	# Docker: "linux/arm64",   Debian: "arm64"
	GOARCH=arm64            $(GO_BUILD) -o _artifacts/repro-get-$(VERSION).linux-arm64   ./cmd/repro-get
	# Docker: "linux/ppc64le", Debian: "ppc64el
	GOARCH=ppc64le          $(GO_BUILD) -o _artifacts/repro-get-$(VERSION).linux-ppc64le ./cmd/repro-get
	# Docker: "linux/riscv64", Debian: riscv64"
	GOARCH=riscv64          $(GO_BUILD) -o _artifacts/repro-get-$(VERSION).linux-riscv64 ./cmd/repro-get
	# Docker: "linux/s390x",   Debian: s390x"
	GOARCH=s390x            $(GO_BUILD) -o _artifacts/repro-get-$(VERSION).linux-s390x   ./cmd/repro-get
	$(UPX) -o _artifacts/repro-get-$(VERSION).linux-amd64.upx   _artifacts/repro-get-$(VERSION).linux-amd64
	$(UPX) -o _artifacts/repro-get-$(VERSION).linux-arm-v7.upx  _artifacts/repro-get-$(VERSION).linux-arm-v7
	$(UPX) -o _artifacts/repro-get-$(VERSION).linux-arm64.upx   _artifacts/repro-get-$(VERSION).linux-arm64
	$(UPX) -o _artifacts/repro-get-$(VERSION).linux-ppc64le.upx _artifacts/repro-get-$(VERSION).linux-ppc64le
	# No UPX for riscv64 and s390x
	touch -d @$(SOURCE_DATE_EPOCH) ./_artifacts/repro-get-$(VERSION).linux-*

.PHONY: artifacts-misc
artifacts-misc:
	mkdir -p _artifacts
	# TODO: create the go-mod-vendor archive deterministically
	# rm -rf vendor
	# go mod vendor
	# $(TAR_C) -f _artifacts/repro-get-$(VERSION).go-mod-vendor.tar go.mod go.sum vendor
	# touch -d @$(SOURCE_DATE_EPOCH) _artifacts/repro-get-$(VERSION).go-mod-vendor.*
	echo $(VERSION) >_artifacts/VERSION
	echo $(SOURCE_DATE_EPOCH) >_artifacts/SOURCE_DATE_EPOCH
	touch -d @$(SOURCE_DATE_EPOCH) _artifacts/VERSION _artifacts/SOURCE_DATE_EPOCH

.PHONY: artifacts.docker
artifacts.docker:
	$(DOCKER_BUILD) --output=./_artifacts -f Makefile.d/Dockerfile.artifacts .
