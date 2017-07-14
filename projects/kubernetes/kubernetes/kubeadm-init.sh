#!/bin/sh
kubeadm init --skip-preflight-checks --kubernetes-version v1.6.1 && kubectl create -n kube-system -f /etc/weave.yaml
