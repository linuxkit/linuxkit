# Will do a Go build in future
# Tag: 1ae7bf8ec49a6537a93fba0c90720c65fa1c6ece
FROM mobylinux/alpine-build-go@sha256:5e9aed92363c25349c2845b9be4a5285e0f56376b8b3ce92c7361bb59e6eeb2d

COPY test.sh mksh /tmp/bin/
COPY ca-certificates.crt /tmp/etc/ssl/certs/ca-certificates.crt

WORKDIR /tmp

RUN printf 'FROM scratch\nCOPY . ./\nENTRYPOINT ["/bin/sh", "-c"]\nCMD ["/bin/test.sh"]' > Dockerfile

RUN mv bin/mksh bin/sh

CMD ["tar", "cf", "-", "."]
