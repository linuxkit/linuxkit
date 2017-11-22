#!/bin/sh
# copied from projects/compose, with fixes for shellcheck
# shellcheck disable=SC2039
set -e

#########
#
# wait for docker socket to be ready, then run the rest of the command
#
########
RETRIES=${RETRIES:-"-1"}
WAIT=${WAIT:=10}
[ -n "$DEBUG" ] && set -x

# keep retrying until docker is ready or we hit our limit
retry_or_fail() {
  local retry_count=0
  local success=1
  local cmd=$1
  local retryMax=$2
  local retrySleep=$3
  local message=$4
  until [ "$retry_count" -ge "$retryMax" ] && [ "$retryMax" -ne -1 ]; do
    echo "trying to $message"
    set +e
    $cmd
    success=$?
    set -e
    [ $success -eq 0 ] && break
    retry_count=$(( retry_count+1 )) || true
    echo "attempt number $retry_count failed to $message, sleeping $retrySleep seconds..."
    sleep "$retrySleep"
  done
  # did we succeed?
  if [ $success -ne 0 ]; then
    echo "failed to $message after $retryMax tries. Exiting..." >&2
    exit 1
  fi
}

connect_to_docker() {
  [ -S /var/run/docker.sock ] || return 1
  curl --unix-socket /var/run/docker.sock http://localhost/containers/json >/dev/null 2>&1 || return 1
}
# try to connect to docker
retry_or_fail connect_to_docker "$RETRIES" "$WAIT" "connect to docker"

# if we got here, we succeeded
exec "$@"
