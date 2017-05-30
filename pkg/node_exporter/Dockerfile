FROM prom/node-exporter:v0.14.0@sha256:ea8396dd32be6195ef03438b2f459b2e69a88b8fc798c3fc51559f56d4b158fa

ENTRYPOINT ["/bin/node_exporter", "-collector.procfs",  "/host/proc", \
            "-collector.sysfs",  "/host/sys", \
            "-collector.filesystem.ignored-mount-points", \
            "^/(sys|proc|dev|host|etc)($|/)"]
LABEL org.mobyproject.config='{"pid": "host", "binds": ["/proc:/host/proc", "/sys:/host/sys", "/:/rootfs"], "capabilities": ["all"]}'
