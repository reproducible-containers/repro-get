BASE_IMAGE ?= debian:bullseye-20211220
REPRO_GET ?= $(shell command -v repro-get)
REPRO_GET_FLAGS := --distro=debian

DOCKER ?= docker
DOCKER_BUILD ?= $(DOCKER) build

BUILDARCH := $(shell uname -m | sed -e s@x86_64@amd64@ -e s@aarch64@arm64@ -e s@armv7l@arm/v7@)
PLATFORMS ?= linux/$(BUILDARCH)

define gen_simple
	$(REPRO_GET) $(REPRO_GET_FLAGS) dockerfile generate ./$(1) $(BASE_IMAGE)
endef

define gen_full
	$(REPRO_GET) $(REPRO_GET_FLAGS) dockerfile generate ./$(1) $(BASE_IMAGE) $(shell cat ./$(1)/PACKAGES)
	cp -f $(REPRO_GET) ./$(1)/repro-get.linux-$(BUILDARCH)
	$(DOCKER_BUILD) --output type=local,dest=./$(1) --platform=$(PLATFORMS) -f ./$(1)/Dockerfile.generate-hash ./$(1)
	rm -f ./$(1)/repro-get.linux-$(BUILDARCH)
	$(REPRO_GET) $(REPRO_GET_FLAGS) dockerfile generate ./$(1) $(BASE_IMAGE)
endef

all: hello gcc

.PHONY: hello
hello:
	# In this example, hash files are simple enough to maintain manually
	$(call gen_simple,$@)

.PHONY: gcc
gcc:
	$(call gen_full,$@)
