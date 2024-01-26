FROM linuxkit/alpine:6584375f70524a4684f10e7155d91380420f5392 AS build
RUN apk add --no-cache --initdb make

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /usr/bin/make /usr/bin/
COPY infile infile
