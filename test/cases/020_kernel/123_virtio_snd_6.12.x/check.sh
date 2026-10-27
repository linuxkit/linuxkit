#!/bin/sh
set -eu

# No sound device is required: this checks the packaged module and dependencies.
uname -a
modinfo virtio_snd
modprobe virtio_snd
test -d /sys/module/virtio_snd
test -d /sys/module/snd_pcm
test -d /sys/module/snd
test -r /proc/asound/cards
cat /proc/asound/cards

printf 'Virtio sound module test PASSED\n'
