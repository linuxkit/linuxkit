#!/bin/bash

set -o errexit
set -o pipefail
set -o nounset

ns="test-hostport-$$"

enum=($(seq 1 10))

str="TEST-$$-$(date +%s)"

pods_ready() {
  kubectl get pods --output="jsonpath={range .items[0]}{.status.phase}{end}" "$@" | grep -q Running
}

kubectl create namespace "${ns}"

kubectl run server --namespace="${ns}" --image=alpine -- nc -p 2020 -lk -e echo "${str}"

for i in "${enum[@]}" ; do
  kubectl expose deployment server --namespace="${ns}" --name="server-${i}" --port=2020 --target-port=2020 --type=NodePort
done

until pods_ready --namespace="${ns}" --selector="run=server" ; do sleep 1 ; done

ports=($(kubectl get services --namespace="${ns}" --output="jsonpath={range .items[*]}{.spec.ports[0].nodePort} {end}"))

for p in "${ports[@]}" ; do
  nc localhost "${p}" | grep -q "${str}"
done

kubectl delete namespace "${ns}"
