FROM mobylinux/alpine-build-go:e726e12b5eea95a4e7aa537e416139d52040c68b
COPY infrakit/ /go/src/github.com/docker/infrakit
WORKDIR /go/src/github.com/docker/infrakit
ARG VERSION
ARG REVISION
ENV CGO_ENABLED=0
RUN make VERSION=${VERSION} REVISION=${REVISION} binaries
CMD cd build && tar cf - infrakit*
