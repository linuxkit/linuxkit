#!/bin/sh

AUDIT_HASH=59763dd8e587d1821f2d039b2bf446c3a31ea58e

set -e

cd /home/builder

git clone https://github.com/alpinelinux/aports && cd aports && git checkout $AUDIT_HASH
cd testing/audit

abuild-keygen -a
abuild -F -r

find ~/packages
cp ~/packages/testing/$(abuild -A)/*apk ~
