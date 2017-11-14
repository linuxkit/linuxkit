FROM prom/node-exporter:v0.15.1@sha256:88c602650e861bb8045ac83421edecbd056fc58708d47554a0d1a10f19ed7556

ENTRYPOINT ["/bin/node_exporter", "--path.procfs",  "/host/proc", \
            "--path.sysfs",  "/host/sys", \
            "--collector.filesystem.ignored-mount-points", \
            "^/(sys|proc|dev|host|etc)($|/)"]
LABEL org.mobyproject.config='{"pid": "host", "binds": ["/proc:/host/proc", "/sys:/host/sys", "/:/rootfs"], "capabilities": ["all"]}'
