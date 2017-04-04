# A dockerfile to build an etcd container image from the upstream one
# with a script as entry point
FROM quay.io/coreos/etcd:v3.1.5
COPY ./etcd.sh /

ENTRYPOINT ["/etcd.sh"]
