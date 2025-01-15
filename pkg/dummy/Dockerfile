FROM linuxkit/alpine:e9313150d2cf07ad92b0adb86799dedd9a0476e4 AS build
RUN apk add --no-cache --initdb make

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /usr/bin/make /usr/bin/
COPY infile infile
