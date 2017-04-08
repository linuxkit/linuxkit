# A dockerfile to build an etcd container image from the upstream one
# with a script as entry point
FROM quay.io/coreos/etcd:v3.1.5

# sfdisk and jq re required to mount the disk
RUN apk add --no-cache sfdisk jq

COPY ./etcd.sh /

ENTRYPOINT ["/etcd.sh"]
