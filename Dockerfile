FROM golang:1.20.0-bullseye@sha256:c04e94c05b90347616893cfe487cce0581904e46289a869ad871e1e50ec8b43d AS build-artifacts
COPY . /src
WORKDIR /src

# Install repro-get
RUN --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go \
  make && \
  make install

# Install upx with repro-get
ARG BUILDARCH
ARG BUILDVARIANT
RUN --mount=type=cache,target=/var/cache/repro-get \
  echo "8210713cc4f21b63e666e395d61d4da0ec77ee0d000ce9f31f83b05fa4d78429  pool/main/u/ucl/libucl1_1.03+repack-5_amd64.deb" >> /tmp/SHA256SUMS-amd64 && \
  echo "fe4ee9d376ee5008319ce0d9f70bf8608d7240694ad4842582d406caad23f130  pool/main/u/upx-ucl/upx-ucl_3.95-1_amd64.deb"    >> /tmp/SHA256SUMS-amd64 && \
  repro-get install /tmp/SHA256SUMS-${BUILDARCH}${BUILDVARIANT:+-${BUILDVARIANT}}

# Build all the artifacts
RUN --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go \
  make artifacts

FROM scratch AS artifacts
COPY --from=build-artifacts /src/_artifacts/ /

FROM artifacts
CMD ["echo", "This is an artifact-only image. Not runnable with `docker run`."]
