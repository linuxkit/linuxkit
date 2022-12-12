# this FROM *must* be updated when changing ../build1; just run `lkt pkg show-tag ../build1` and place it here
FROM --platform=linux/amd64 linuxkit/chained-build-platform-image1-test-does-not-exist-anywhere-else-123456789:8764ad5a2481b9c6b213a0cea6abaa30085481be as src
FROM --platform=linux/amd64 alpine:3.15
COPY --from=src /foo /foo
RUN cat /foo
# copy output of uname commands from first build and compare with current
COPY --from=src /uname* /
RUN uname -m >/uname_m_current && uname -s >/uname_s_current && cat /uname_m_current && cat /uname_s_current
RUN diff /uname_m /uname_m_current && diff /uname_s /uname_s_current