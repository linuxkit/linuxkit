#!/bin/sh

# script to generate the iso image we use here
set -ex

apk --update add cdrkit >&2

# generate our cdrom image
cd /tmp;
{ echo instance-id: iid-local01; echo local-hostname: cloudimg; } > meta-data
printf '{"file1": {"perm": "0644","content": "abcdefg"}, "file2": {"perm": "0700", "content": "supersecret"}}' > user-data
genisoimage  -output - -volid cidata -joliet -rock user-data meta-data
