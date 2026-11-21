FROM linuxkit/alpine:d497d5be5b78bfd78268f3466cc4080aee5d7389 AS build
RUN apk add --no-cache --initdb make

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /usr/bin/make /usr/bin/
COPY infile infile
