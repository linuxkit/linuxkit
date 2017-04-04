#! /bin/sh
##
## This script starts a etcd container which is used to bootstrap a
## local etcd cluster. The etcd container is started on a non-standard
## port to keep the standard port free for the cluster.
##
## If you have a local etcd installed (brew install etcd) you can
## point the cli at it as well:
##
## etcdctl --debug --endpoints http://0.0.0.0:2381 member list
##

# debug
set -x
set -v

# Change depending on the cluster size
NUMPEERS=1

# Start a local etcd for bootstrapping
NAME=etcd-bootstrap
PORT=2381

#UUID=$(uuidgen)
UUID=6c007a14875d53d9bf0ef5a6fc0257c817f0fb83

ID=$(docker run -d --rm --name ${NAME} \
     -p ${PORT}:${PORT} \
     quay.io/coreos/etcd:v3.1.5 /usr/local/bin/etcd \
       --debug \
       --name ${NAME} \
       --listen-client-urls http://0.0.0.0:${PORT} \
       --advertise-client-urls http://0.0.0.0:$PORT,http://192.168.65.2:$PORT \
       --initial-cluster-token ${NAME} \
       --initial-cluster-state new \
       --auto-compaction-retention 0)

trap "docker kill ${ID}" 2

# Could poll until returns without error, but sleep for 2s for now
sleep 2
docker exec -t ${ID} etcdctl --endpoints http://0.0.0.0:${PORT} mk discovery/${UUID}/_config/size ${NUMPEERS}

echo "KEY: ${UUID}"

docker logs -f ${ID}
