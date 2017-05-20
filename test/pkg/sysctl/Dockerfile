FROM alpine:edge
ADD . ./
ENTRYPOINT ["/bin/sh", "/check.sh"]
LABEL org.mobyproject.config='{"net": "host","pid": "host", "ipc": "host", "readonly": true}'
