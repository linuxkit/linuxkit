FROM linuxkit/alpine:8382c70bfe6b581d2c6b3032ca1081ef2ef76e1f AS build
RUN apk add --no-cache --initdb make

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /usr/bin/make /usr/bin/
COPY infile infile
