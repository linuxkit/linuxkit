# Tag: b77cfc4ad0033d4366df830ed697afc7bab458a2
FROM mobylinux/alpine-build-c@sha256:53739ea6042cb0ac39cf6e262012c1c4224206b2c9b719569fe7efa3a381348c

COPY . /

ENV VERSION=mksh-R54

RUN curl -O -sSL https://github.com/MirBSD/mksh/archive/$VERSION.tar.gz
RUN zcat $VERSION.tar.gz | tar xvf -

WORKDIR mksh-$VERSION

ENV LDFLAGS=-static

RUN sh ./Build.sh
RUN strip mksh
RUN install -c -s -o root -g bin -m 555 mksh /bin/mksh
