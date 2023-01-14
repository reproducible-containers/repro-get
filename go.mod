module github.com/reproducible-containers/repro-get

go 1.19

require (
	github.com/cheggaaa/pb/v3 v3.1.0
	github.com/containerd/containerd v1.6.8 // replaced
	github.com/containerd/continuity v0.3.0
	github.com/containerd/nerdctl v0.23.1-0.20221008120401-b4c01094b581
	github.com/cyphar/filepath-securejoin v0.2.3
	github.com/fatih/color v1.13.0
	github.com/google/go-cmp v0.5.9
	github.com/mattn/go-isatty v0.0.17
	github.com/opencontainers/go-digest v1.0.0
	github.com/sirupsen/logrus v1.9.0
	github.com/spf13/cobra v1.5.0
	gotest.tools/v3 v3.4.0
	pault.ag/go/debian v0.12.0
)

require (
	github.com/AdaLogics/go-fuzz-headers v0.0.0-20221007124625-37f5449ff7df // indirect
	github.com/Microsoft/go-winio v0.6.0 // indirect
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/docker/cli v20.10.18+incompatible // indirect
	github.com/docker/docker v20.10.18+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.7.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/klauspost/compress v1.15.11 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc2 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rivo/uniseg v0.4.2 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/crypto v0.0.0-20221005025214-4161e89ecf1b // indirect
	golang.org/x/mod v0.6.0-dev.0.20220419223038-86c51ed26bb4 // indirect
	golang.org/x/sync v0.0.0-20220929204114-8fcdb60fdcc0 // indirect
	golang.org/x/sys v0.0.0-20221006211917-84dc82d7e875 // indirect
	golang.org/x/tools v0.1.12 // indirect
	google.golang.org/genproto v0.0.0-20220930163606-c98284e70a91 // indirect
	google.golang.org/grpc v1.50.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	pault.ag/go/topsort v0.1.1 // indirect
)

// https://github.com/containerd/containerd/pull/7460
replace github.com/containerd/containerd => github.com/AkihiroSuda/containerd v1.7.0-prealpha.202208060102.0.20221007084504-a15d6d66f390
