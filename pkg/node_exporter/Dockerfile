FROM prom/node-exporter:v0.14.0

ENTRYPOINT ["/bin/node_exporter", "-collector.procfs",  "/host/proc", \
            "-collector.sysfs",  "/host/sys", \
            "-collector.filesystem.ignored-mount-points", \
            "^/(sys|proc|dev|host|etc)($|/)"]
