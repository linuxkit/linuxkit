FROM linuxkit/alpine:35b33c6b03c40e51046c3b053dd131a68a26c37a AS build
RUN apk add --no-cache --initdb make

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /usr/bin/make /usr/bin/
COPY infile infile
