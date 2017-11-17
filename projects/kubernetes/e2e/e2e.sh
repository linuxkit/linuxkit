#!/bin/sh

# cleanup resources created by previous runs
kubectl get namespaces \
  --output="jsonpath={range .items[?(.status.phase == \"Active\")]}{.metadata.name}{\"\n\"}{end}" \
  | grep '^e2e-.*' \
  | xargs -r kubectl delete namespaces

skip="${GINKOGO_SKIP:-'\[Slow\]|\[Serial\]|\[Disruptive\]|\[Flaky\]|\[Feature:.+\]|\[HPA\]|Dashboard|Services.*functioning.*NodePort|.*NFS.*|.*Volume.*|\[sig-storage\]|.*StatefulSet.*|should\ proxy\ to\ cadvisor\ using\ proxy\ subresource'}"
nodes="${GINKOGO_NODES:-"4"}" \
flakeAttempts="${GINKOGO_FLAKE_ATTEMPTS:-"2"}" \

provider="${KUBE_CLOUD_PROVIDER:-"local"}" \
host="https://${KUBE_APISERVER_ADDR:-"192.168.65.2:6443"}" \

# execute the test suite
exec /usr/bin/ginkgo \
  -progress \
  -nodes="${nodes}" \
  -flakeAttempts="${flakeAttempts}" \
  -skip="${skip}" \
  /usr/bin/e2e.test -- \
    -provider="${provider}" \
    -kubeconfig=/etc/kubernetes/admin.conf \
    -host="${host}" \
    -test.short \
    -test.v
