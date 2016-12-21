# Will do a Go build in future
# Tag: d3f266a146a79f35d3bebf252cce62eee25fcfa9
FROM mobylinux/alpine-build-go@sha256:1eca9f912cfa4f59ad817acad76744516b999395c080bc01a50b3a2b3a9a3f5c

COPY test.sh mksh /tmp/bin/
COPY ca-certificates.crt /tmp/etc/ssl/certs/ca-certificates.crt

WORKDIR /tmp

RUN printf 'FROM scratch\nCOPY . ./\nENTRYPOINT ["/bin/sh", "-c"]\nCMD ["/bin/test.sh"]' > Dockerfile

RUN mv bin/mksh bin/sh

CMD ["tar", "cf", "-", "."]
