# those IMAGE and TAG *must* be updated when changing ../build1; just run `lkt pkg show-tag ../build1` and place it here
ARG IMAGE=linuxkit/chained-build-with-args-image1-test-does-not-exist-anywhere-else-123456789
ARG TAG=6a40ba0866b7ce18b3b13390bf88bfeb6ac0db4e
FROM ${IMAGE}:${TAG} as src
FROM alpine:3.15
COPY --from=src /foo /foo
RUN cat /foo