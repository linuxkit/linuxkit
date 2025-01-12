FROM linuxkit/alpine:8c02670fd2e5c28d03fd39db0bddc1379b3af6b2 AS build
RUN apk add --no-cache --initdb make

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /usr/bin/make /usr/bin/
COPY infile infile
