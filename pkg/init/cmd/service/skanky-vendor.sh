#!/bin/sh
#
# We also need the containerd client and its transitive dependencies
# and we conveniently have a checkout already. We actually prefer to
# reuse containerd's vendoring for consistency anyway.

set -eu
ctrd=$1
cp -r $ctrd/vendor/* vendor/
# We need containerd itself of course
mkdir -p vendor/github.com/containerd
cp -r $ctrd vendor/github.com/containerd/containerd
# Stop go finding nested vendorings
rm -rf vendor/github.com/containerd/containerd/vendor
