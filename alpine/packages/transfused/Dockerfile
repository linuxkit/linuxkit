FROM mobylinux/alpine-build-c

RUN mkdir -p /transfused
WORKDIR /transfused

COPY . /transfused

RUN make transfused
