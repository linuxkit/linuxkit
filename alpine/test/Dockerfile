FROM alpine:3.5

COPY test.sh mksh /tmp/bin/
COPY ca-certificates.crt /tmp/etc/ssl/certs/ca-certificates.crt
COPY moby /tmp/etc/

WORKDIR /tmp

RUN printf 'FROM scratch\nCOPY . ./\nENTRYPOINT ["/bin/sh", "-c"]\nCMD ["/bin/test.sh"]' > Dockerfile

RUN mv bin/mksh bin/sh

CMD ["tar", "cf", "-", "."]
