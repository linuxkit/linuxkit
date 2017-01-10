#!/bin/sh

# tag: 801f33408e43e6a22985aa994ab0bcba41659ec6
RIDDLER=mobylinux/riddler@sha256:2dda30eb24ac531a9f2164e9592a21538b5841f2ca8459b0c190da46ea7dfafd

docker run --rm -v /var/run/docker.sock:/var/run/docker.sock $RIDDLER "$@"
