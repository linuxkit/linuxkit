#!/bin/sh
set -e
kubeadm init --skip-preflight-checks --kubernetes-version @KUBERNETES_VERSION@
kubectl create -n kube-system -f /etc/weave.yaml
