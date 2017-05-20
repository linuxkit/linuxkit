FROM alpine:3.5
ADD . ./
ENTRYPOINT ["/bin/sh", "/poweroff.sh"]
LABEL org.mobyproject.config='{"net": "host","pid": "host", "ipc": "host", "readonly": true, "capabilities": ["CAP_SYS_BOOT"]}'
