#!/bin/sh

# save the input to /file.img
cat > /file.img

echo "mtools_skip_check=1" >> /etc/mtools.conf

exec $@
