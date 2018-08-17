#!/bin/sh
set -e

#########
#
# load any cached mounted images, and run compose
#
########

[ -n "$DEBUG" ] && set -x

if [ -d /compose/images/ ]; then
	for image in /compose/images/*.tar ; do
		docker image load -i $image
	done
fi


docker-compose -f /compose/docker-compose.yml up -d
