FROM linuxkit/alpine:6090baae063eb5023c9601966e88df831f789a70 AS build
RUN apk add --no-cache --initdb make

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /usr/bin/make /usr/bin/
COPY infile infile
