FROM linuxkit/alpine:7aea0bb7faf70f35c6b7d2d244c91da1d4b3ea93 AS build
RUN apk add --no-cache --initdb make

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /usr/bin/make /usr/bin/
COPY infile infile
