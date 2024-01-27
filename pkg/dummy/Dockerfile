FROM linuxkit/alpine:29e84088fdd519268b05c622e4437d205acb6323 AS build
RUN apk add --no-cache --initdb make

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /usr/bin/make /usr/bin/
COPY infile infile
