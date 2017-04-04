#! /bin/sh

# debug
set -x
set -v

# Needs to come from metadata
UUID=6c007a14875d53d9bf0ef5a6fc0257c817f0fb83
DISCOVER_URL=http://192.168.65.2:2381/v2/keys/discovery/${UUID}

IP=$(ifconfig eth0 2>/dev/null|awk '/inet addr:/ {print $2}'|sed 's/addr://')
NAME=$(hostname)

/usr/local/bin/etcd \
    --name ${NAME} \
    --debug \
    --log-package-levels etcdmain=DEBUG,etcdserver=DEBUG \
    --initial-advertise-peer-urls http://${IP}:2380 \
    --listen-peer-urls http://${IP}:2380 \
    --listen-client-urls http://${IP}:2379,http://127.0.0.1:2379 \
    --advertise-client-urls http://${IP}:2379 \
    --discovery ${DISCOVER_URL}
