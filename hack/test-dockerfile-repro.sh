#!/bin/bash
set -eu -o pipefail

: "${DOCKER:=docker}"
: "${DOCKER_RUN:=${DOCKER} run}"

# BuildKit needs to be v0.11 or later
: "${BUILDCTL:=buildctl}"
: "${BUILDCTL_BUILD:=${BUILDCTL} build}"

# diffoscope can't be pinned by its hash, as the registry does not retain old versions
: "${DIFFOSCOPE_IMAGE:=registry.salsa.debian.org/reproducible-builds/diffoscope}"
: "${WORKSPACE:=/tmp/df-repro/$(date +%s)}"

if [ $# -ne 1 ]; then
	echo >&2 "Usage: $0 ."
	exit 1
fi
context_dir="$1"

: "${SOURCE_DATE_EPOCH:=$(cat "${context_dir}/SOURCE_DATE_EPOCH")}"

INFO() {
	set +x
	/bin/echo -e "\e[104m\e[97m[INFO]\e[49m\e[39m ${*}"
	set -x
}

INFO "WORKSPACE: ${WORKSPACE}"
rm -rf "${WORKSPACE}"
mkdir -p "${WORKSPACE}"

b() {
	${BUILDCTL_BUILD} \
		--frontend dockerfile.v0 \
		--local dockerfile="${context_dir}" \
		--local context="${context_dir}" \
		--opt build-arg:SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH}" \
		"$@"
}
INFO "===== Building 2 times ====="
for i in 0 1; do
	b --no-cache --output type=tar,dest="${WORKSPACE}/${i}-raw.tar"
	b --output type=oci,buildinfo=false,dest="${WORKSPACE}/${i}-oci.tar"
done

code=0
for t in raw oci; do
	INFO "===== Testing reproducibility of ${t} tar archives ====="
	sha256sum "${WORKSPACE}/0-${t}.tar" "${WORKSPACE}/1-${t}.tar" | tee "${WORKSPACE}/SHA256SUMS-${t}-tar"
	if [ "$(cut -f1 -d" " <"${WORKSPACE}/SHA256SUMS-${t}-tar" | uniq | wc -l)" == 1 ]; then
		INFO "The ${t} tar archives seem reproducible"
	else
		INFO "The ${t} tar archives DO NOT seem reproducible"
		${DOCKER_RUN} --rm -t -w "${WORKSPACE}" -v "${WORKSPACE}:${WORKSPACE}":ro "${DIFFOSCOPE_IMAGE}" 0-${t}.tar 1-${t}.tar
		code=1
	fi
done
exit "${code}"
