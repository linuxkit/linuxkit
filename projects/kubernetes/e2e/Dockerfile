FROM linuxkit/alpine:3fb44354a34b05134fbf585a00217cd2f8c8f0bf AS build

RUN apk add -U --no-cache \
  curl \
  && true

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    bash \
    ca-certificates \
    curl \
    musl \
    socat \
    util-linux \
    libc6-compat \
    && true
# Remove apk residuals. We have a read-only rootfs, so apk is of no use.
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

RUN rmdir /out/var/run && ln -nfs /run /out/var/run

COPY _cache/kubernetes-test.tar.gz /tmp/kubernetes-test.tar.gz
RUN tar -C /tmp -xzf /tmp/kubernetes-test.tar.gz kubernetes/platforms/linux/amd64/ginkgo kubernetes/platforms/linux/amd64/e2e.test && \
    mv /tmp/kubernetes/platforms/linux/amd64/ginkgo /out/usr/bin/ginkgo && \
    mv /tmp/kubernetes/platforms/linux/amd64/e2e.test /out/usr/bin/e2e.test
COPY _cache/kubectl /out/usr/bin/kubectl
RUN chmod 0755 /out/usr/bin/kubectl

ADD e2e.sh /out/usr/bin/e2e.sh

FROM scratch
WORKDIR /
ENTRYPOINT ["/usr/bin/e2e.sh"]
COPY --from=build /out /
ENV KUBECONFIG "/etc/kubernetes/admin.conf"
