#!/bin/sh
set -e

#########
#
# load any cached mounted images, and run compose
#
########

[ -n "$DEBUG" ] && set -x

for image in /compose/images/*.tar ; do
	docker image load -i $image && rm -f $image
done


docker-compose -f /compose/docker-compose.yml up -d
