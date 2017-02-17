FROM golang:1.8-alpine

RUN \
  apk update && apk upgrade && \
  apk add \
  docker \
  gcc \
  git \
  jq \
  linux-headers \
  musl-dev \
  tar \
  && true

COPY Dockerfile /
COPY riddler.sh /usr/bin/

RUN git clone https://github.com/jessfraz/riddler.git /go/src/github.com/jessfraz/riddler

WORKDIR /go/src/github.com/jessfraz/riddler
RUN git checkout 23befa0b232877b5b502b828e24161d801bd67f6
RUN go build -o /usr/bin/riddler .

ENTRYPOINT ["/usr/bin/riddler.sh"]
