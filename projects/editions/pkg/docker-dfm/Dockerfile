FROM linuxkit/slirp-proxy:f1eec56e1a7196f6a786223baa30e2f5e026a27d as slirp-proxy

FROM docker:17.05.0-ce-dind
COPY --from=slirp-proxy /slirp-proxy /usr/bin/slirp-proxy
RUN addgroup -S docker
CMD [ "/usr/local/bin/dockerd", "-H", "unix:///var/run/docker.sock", "--swarm-default-advertise-addr=eth0", "--userland-proxy-path", "/usr/bin/slirp-proxy", "--storage-driver", "overlay2", "--log-driver", "cpuguy83/memlogd-docker"]
