FROM mobylinux/alpine-build-c

RUN mkdir -p /hvtools

COPY src /hvtools/
RUN tar czvf /hvtools.tar.gz hvtools

WORKDIR /hvtools
RUN make
