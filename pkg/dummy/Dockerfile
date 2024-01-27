FROM linuxkit/alpine:275ef778d89133ebec5e80d82c664c3549f919d4 AS build
RUN apk add --no-cache --initdb make

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /usr/bin/make /usr/bin/
COPY infile infile
