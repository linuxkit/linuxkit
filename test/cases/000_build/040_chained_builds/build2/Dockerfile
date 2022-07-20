# this FROM *must* be updated when changing ../build1; just run `lkt pkg show-tag ../build1` and place it here
FROM linuxkit/chained-build-image1-test-does-not-exist-anywhere-else-123456789:ad06eb29c48f0daab0cc2836a1351a053948100c as src
FROM alpine:3.15
COPY --from=src /foo /foo
RUN cat /foo