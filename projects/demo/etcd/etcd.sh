#! /bin/sh

# debug
set -x
set -v

# Wait till we have an IP address
IP=""
while [ -z "$IP" ]; do
    IP=$(ifconfig eth0 2>/dev/null|awk '/inet addr:/ {print $2}'|sed 's/addr://')
    sleep 1
done

# Name is infra+last octet of IP address
NUM=$(echo ${IP} | cut -d . -f 4)
PREFIX=$(echo ${IP} | cut -d . -f 1,2,3)
NAME=infra${NUM}

# This should come from Metadata
INIT_CLUSTER=infra200=http://${PREFIX}.200:2380,infra201=http://${PREFIX}.201:2380,infra202=http://${PREFIX}.202:2380,infra203=http://${PREFIX}.203:2380,infra204=http://${PREFIX}.204:2380

# Try to start in *new* cluster mode
/usr/local/bin/etcd \
    --name ${NAME} \
    --debug \
    --log-package-levels etcdmain=DEBUG,etcdserver=DEBUG \
    --initial-advertise-peer-urls http://${IP}:2380 \
    --listen-peer-urls http://${IP}:2380 \
    --listen-client-urls http://${IP}:2379,http://127.0.0.1:2379 \
    --advertise-client-urls http://${IP}:2379 \
    --initial-cluster-token etcd-cluster-1 \
    --initial-cluster ${INIT_CLUSTER} \
    --initial-cluster-state new

[ $? -eq 0 ] && exit 0

# Joining the new cluster failed. Let's try joining an *existing* cluster
/usr/local/bin/etcd \
    --name ${NAME} \
    --debug \
    --log-package-levels etcdmain=DEBUG,etcdserver=DEBUG \
    --initial-advertise-peer-urls http://${IP}:2380 \
    --listen-peer-urls http://${IP}:2380 \
    --listen-client-urls http://${IP}:2379,http://127.0.0.1:2379 \
    --advertise-client-urls http://${IP}:2379 \
    --initial-cluster ${INIT_CLUSTER} \
    --initial-cluster-state existing
