FROM linuxkit/alpine:7f3944798557de5518a56e3437d7ed982701f224 AS build
RUN apk add --no-cache --initdb make

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /usr/bin/make /usr/bin/
COPY infile infile
