#!/bin/bash
set -eu -o pipefail

: "${DOCKER:=docker}"
: "${DOCKER_RUN:=${DOCKER} run}"

: "${BUILDCTL:=buildctl}"
: "${BUILDCTL_BUILD:=${BUILDCTL} build}"

: "${DIFFOSCOPE_IMAGE:=registry.salsa.debian.org/reproducible-builds/diffoscope@sha256:e08b04b39228abc9d9a3f59004430b110d812be793cd8c682355bea2b0398a61}" # 2022-09-23
: "${WORKSPACE:=/tmp/df-repro/$(date +%s)}"

if [ $# -ne 1 ]; then
	echo >&2 "Usage: $0 ."
	exit 1
fi
context_dir="$1"

INFO() {
	set +x
	/bin/echo -e "\e[104m\e[97m[INFO]\e[49m\e[39m ${*}"
	set -x
}

INFO "WORKSPACE: ${WORKSPACE}"
rm -rf "${WORKSPACE}"
mkdir -p "${WORKSPACE}"

INFO "===== Building 2 times ====="
for i in 0 1; do
	${BUILDCTL_BUILD} --no-cache --output type=tar,dest="${WORKSPACE}/${i}-raw.tar" \
		--frontend dockerfile.v0 --local dockerfile="${context_dir}" --local context="${context_dir}"
	${BUILDCTL_BUILD} --output type=oci,dest="${WORKSPACE}/${i}-oci.tar,annotation-manifest-descriptor.org.opencontainers.image.created=1970-01-01T00:00:00Z" \
		--frontend dockerfile.v0 --local dockerfile="${context_dir}" --local context="${context_dir}"
done

INFO "NOTE: Reproduction of raw tar needs https://github.com/moby/buildkit/pull/3149 to be merged"
INFO "NOTE: Reproduction of oci tar needs https://github.com/moby/buildkit/pull/3152 , https://github.com/moby/buildkit/pull/2918 , https://github.com/containerd/containerd/pull/7478 , and maybe much more work"
for t in raw oci; do
	INFO "===== Testing reproducibility of ${t} tar archives ====="
	sha256sum "${WORKSPACE}/0-${t}.tar" "${WORKSPACE}/1-${t}.tar" | tee "${WORKSPACE}/SHA256SUMS-${t}-tar"
	if [ "$(cut -f1 -d" " <"${WORKSPACE}/SHA256SUMS-${t}-tar" | uniq | wc -l)" == 1 ]; then
		INFO "The ${t} tar archives seem reproducible"
	else
		INFO "The ${t} tar archives DO NOT seem reproducible"
		${DOCKER_RUN} --rm -t -w "${WORKSPACE}" -v "${WORKSPACE}:${WORKSPACE}":ro "${DIFFOSCOPE_IMAGE}" 0-${t}.tar 1-${t}.tar
	fi
done
