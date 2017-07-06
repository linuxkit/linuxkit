#!/bin/sh
set -e
if [ $# -ne 2 ] ; then
    echo "Need <OLD> and <NEW> as arguments" >&2
    exit 1
fi
old=$1
new=$2

git grep -l "$old" | xargs sed -i -e "s,$old,$new,g"
