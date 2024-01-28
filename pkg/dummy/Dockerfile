FROM linuxkit/alpine:146f540f25cd92ec8ff0c5b0c98342a9a95e479e AS build
RUN apk add --no-cache --initdb make

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /usr/bin/make /usr/bin/
COPY infile infile
