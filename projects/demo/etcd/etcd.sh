#! /bin/sh

# debug
set -x
set -v

# --initial-cluster argument should come from meta data

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

/usr/local/bin/etcd \
    --name ${NAME} \
    --debug \
    --log-package-levels etcdmain=DEBUG,etcdserver=DEBUG \
    --initial-advertise-peer-urls http://${IP}:2380 \
    --listen-peer-urls http://${IP}:2380 \
    --listen-client-urls http://${IP}:2379,http://127.0.0.1:2379 \
    --advertise-client-urls http://${IP}:2379 \
    --initial-cluster-token etcd-cluster-1 \
    --initial-cluster infra200=http://${PREFIX}.200:2380,infra201=http://${PREFIX}.201:2380,infra202=http://${PREFIX}.202:2380 \
    --initial-cluster-state new
