# Will do a Go build in future
# Tag: 2c9434f1c4ff70b102f34a97d2df1a8363a11a65
FROM mobylinux/alpine-build-go@sha256:d528bbf7102e4209bd59ef030d41de9003ab8e42c303956f62b2df47f3e17849

COPY test.sh mksh /tmp/bin/
COPY ca-certificates.crt /tmp/etc/ssl/certs/ca-certificates.crt

WORKDIR /tmp

RUN printf 'FROM scratch\nCOPY . ./\nENTRYPOINT ["/bin/sh", "-c"]\nCMD ["/bin/test.sh"]' > Dockerfile

RUN mv bin/mksh bin/sh

CMD ["tar", "cf", "-", "."]
