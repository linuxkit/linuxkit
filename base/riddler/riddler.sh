#!/bin/sh

set -e

# riddler always adds the apparmor options if this is not present
EXTRA_OPTIONS="--security-opt apparmor=unconfined"

ARGS="$@"
CONTAINER=$(docker create $EXTRA_OPTIONS $ARGS)
riddler $CONTAINER > /dev/null
docker rm $CONTAINER > /dev/null

# unfixed known issues
# noNewPrivileges is always set by riddler, but that is fine for our use cases

# These fixes should be removed when riddler is fixed
# process.rlimits, just a constant at present, not useful
# memory swappiness is too big by default
# remove user namespaces
# --read-only sets /dev ro
# /sysfs ro unless privileged - cannot detect so will do if grant all caps
# 
cat config.json | \
  jq 'del(.process.rlimits)' | \
  jq 'del (.linux.resources.memory.swappiness)' | \
  jq 'del(.linux.uidMappings) | del(.linux.gidMappings) | .linux.namespaces = (.linux.namespaces|map(select(.type!="user")))' | \
  jq 'if .root.readonly==true then .mounts = (.mounts|map(if .destination=="/dev" then .options |= .+ ["ro"] else . end)) else . end' | \
  jq '.mounts = if .process.capabilities | length != 38 then (.mounts|map(if .destination=="/sys" then .options |= .+ ["ro"] else . end)) else . end'
