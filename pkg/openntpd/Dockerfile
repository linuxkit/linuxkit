FROM linuxkit/alpine:f0169b60fb260d74025496ae6fd93213fecaba8f@sha256:23743c7206ebe8a609442c5ac7084a26ed45ce8f5213960428bca264225849f1 AS mirror

FROM alpine:3.5@sha256:dfbd4a3a8ebca874ebd2474f044a0b33600d4523d03b0df76e5c5986cb02d7e8
COPY --from=mirror /etc/apk/repositories /etc/apk/repositories
COPY --from=mirror /etc/apk/keys /etc/apk/keys/
COPY --from=mirror /mirror /mirror/

RUN apk add --no-cache openntpd && true
RUN rm -rf /mirror /etc/apk/repositories /etc/apk/keys

COPY . .

CMD ["/usr/sbin/ntpd", "-d", "-s"]
