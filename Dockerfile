# syntax=docker/dockerfile:1

ARG GO_VERSION=1.19.2
ARG XX_VERSION=1.1.2
ARG OSXCROSS_VERSION=12.3-r0-alpine

# xx is a helper for cross-compilation
FROM --platform=$BUILDPLATFORM tonistiigi/xx:${XX_VERSION} AS xx

# osxcross contains the MacOSX cross toolchain for xx
FROM crazymax/osxcross:${OSXCROSS_VERSION} AS osxcross

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine3.16 AS base
COPY --link --from=xx / /
RUN apk add --no-cache clang git lld llvm make
ARG TARGETPLATFORM
WORKDIR /src
COPY --link src/cmd/linuxkit/ .

FROM base AS build-linux
RUN --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod <<EOT
  set -ex
  xx-go --wrap
  LOCAL_TARGET=/out/linuxkit make local-build
  xx-verify /out/linuxkit
EOT

FROM base AS build-darwin
RUN --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,from=osxcross,src=/osxsdk,target=/xx-sdk <<EOT
  set -ex
  xx-go --wrap
  export CGO_CFLAGS=--target=${TARGETPLATFORM}-apple-macos12.3
  LOCAL_TARGET=/out/linuxkit make local-build
  xx-verify /out/linuxkit
EOT

FROM base AS build-windows
RUN --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod <<EOT
  set -ex
  xx-go --wrap
  LOCAL_TARGET=/out/linuxkit.exe make local-build
  xx-verify /out/linuxkit.exe
EOT

FROM build-$TARGETOS AS build

FROM scratch as binaries
COPY --link --from=build /out /