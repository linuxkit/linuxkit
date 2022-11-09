# syntax=docker/dockerfile:1

# xx is a helper for cross-compilation
FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.1.2 AS xx

# osxcross contains the MacOSX cross toolchain for xx
FROM crazymax/osxcross:12.3-r0-alpine AS osxcross

FROM --platform=$BUILDPLATFORM golang:1.19.2-alpine3.16 AS build
COPY --link --from=xx / /
RUN apk add --no-cache clang git lld llvm make
ARG TARGETPLATFORM
WORKDIR /src
RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,from=osxcross,src=/osxsdk,target=/xx-sdk <<EOT
  set -ex
  xx-go --wrap
  LOCAL_TARGET=/out/linuxkit make -C ./src/cmd/linuxkit local-build
  xx-verify /out/linuxkit
  [ "$(xx-info os)" = "windows" ] && mv /out/linuxkit /out/linuxkit.exe || true
EOT

FROM scratch as binaries
COPY --link --from=build /out /
