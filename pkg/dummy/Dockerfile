FROM linuxkit/alpine:5ac048802293564a50414394e1b9896eac528e7d AS build
RUN apk add --no-cache --initdb make

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /usr/bin/make /usr/bin/
COPY infile infile
