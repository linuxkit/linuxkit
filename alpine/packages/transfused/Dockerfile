FROM mobylinux/alpine-build-c:1b53976210ba819823eda5b1d0d3ea337772e97b

RUN mkdir -p /transfused
WORKDIR /transfused

COPY . /transfused

RUN make transfused
