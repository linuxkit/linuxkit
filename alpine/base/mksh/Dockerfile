# Tag: 9e9c6b252d2e4ec03da23c8f48f54fce7ddee8d3
FROM mobylinux/alpine-build-c@sha256:68737dcc6a1081f07aace1e82aefe90df399a25ae3a025664c4dbdccf76bbd97

COPY . /

ENV VERSION=mksh-R54

RUN curl -O -sSL https://github.com/MirBSD/mksh/archive/$VERSION.tar.gz
RUN zcat $VERSION.tar.gz | tar xvf -

WORKDIR mksh-$VERSION

ENV LDFLAGS=-static

RUN sh ./Build.sh
RUN strip mksh
RUN install -c -s -o root -g bin -m 555 mksh /bin/mksh
