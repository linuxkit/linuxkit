#!/bin/sh
. /etc/init.d/tc-functions

echo "${YELLOW}Running boot2docker shutdown script...${NORMAL}"

/usr/local/etc/init.d/docker stop
