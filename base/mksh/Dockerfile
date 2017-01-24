FROM alpine:3.5
RUN \
  apk update && apk upgrade && \
  apk add \
  curl \
  gcc \
  groff \
  make \
  musl-dev \
  && true

COPY . /

ENV VERSION=mksh-R54

RUN curl -O -sSL https://github.com/MirBSD/mksh/archive/$VERSION.tar.gz
RUN zcat $VERSION.tar.gz | tar xvf -

ENV LDFLAGS=-static

RUN cd mksh-$VERSION && sh ./Build.sh && strip mksh && install -c -s -o root -g bin -m 555 mksh /bin/mksh
