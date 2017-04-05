#!/bin/sh

TIMEOUT=${1:-30}  
sleep "${TIMEOUT}"

/sbin/poweroff -f
