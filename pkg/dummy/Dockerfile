FROM linuxkit/alpine:a720d2b1b83556653abd52d61faa0fe020c2b161 AS build
RUN apk add --no-cache --initdb make

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /usr/bin/make /usr/bin/
COPY infile infile
