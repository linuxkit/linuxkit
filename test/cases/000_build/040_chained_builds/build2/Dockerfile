# this FROM *must* be updated when changing ../build1; just run `lkt pkg show-tag ../build1` and place it here
FROM linuxkit/chained-build-image1-test-does-not-exist-anywhere-else-123456789:e247132d6bc3a56bdce0fcb2d0d4848a1dfcf3b6 as src
FROM alpine:3.15
COPY --from=src /foo /foo
RUN cat /foo
# copy output of uname commands from first build and compare with current
COPY --from=src /uname* /
RUN uname -m >/uname_m_current && uname -s >/uname_s_current && cat /uname_m_current && cat /uname_s_current
RUN diff /uname_m /uname_m_current && diff /uname_s /uname_s_current