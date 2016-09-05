FROM mobylinux/alpine-build-c:1b53976210ba819823eda5b1d0d3ea337772e97b

RUN mkdir -p /hvtools

COPY src /hvtools/
RUN tar czvf /hvtools.tar.gz hvtools

WORKDIR /hvtools
RUN make
