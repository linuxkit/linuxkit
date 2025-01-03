FROM linuxkit/alpine:3fb2ec0c94ed30171bb04100f568bba2ebb648a9 AS build
RUN apk add --no-cache --initdb make

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /usr/bin/make /usr/bin/
COPY infile infile
