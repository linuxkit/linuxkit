#!/bin/sh
/sbin/dmsetup "$@" 2> >(sed 's/device or //g' >&2)
