FROM linuxkit/alpine:5d89cd05a567f9bfbe4502be1027a422d46f4a75 AS build
RUN apk add --no-cache --initdb make

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /usr/bin/make /usr/bin/
COPY infile infile
