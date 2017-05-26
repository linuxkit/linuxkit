#!/bin/bash

set -e

cd /linux && make defconfig

function merge_config()
{
    config=$1
    if [ ! -f "$config" ]; then
        return
    fi

    # A slightly more intelligent merge algorithm: rather than just catting
    # files together (and getting random results), let's explicitly delete the
    # old setting, and then insert our new one.
    while read line; do
        if echo ${line} | grep "is not set" >/dev/null; then
            cfg=$(echo ${line/ is not set/} | cut -c3-)
        else
            cfg=$(echo ${line} | cut -f1 -d=)
        fi

        sed -i -e "/${cfg} is not set/d" -e "/${cfg}=/d" /linux/.config
        echo ${line} >> /linux/.config
    done < "$config"
}

cd /linux && make defconfig && make oldconfig

for config in "$@"; do
  merge_config "$config"
done

cd /linux && make oldconfig

# Let's make sure things are the way we want, i.e. every option we explicitly
# set is set the same way in the resulting config.
function check_config()
{
    if [ ! -f "$1" ]; then return; fi

    while read line; do
      # CONFIG_PANIC_ON_OOPS is special, and set both ways, depending on
      # whether DEBUG is set or not.
      if [ "$line" == *"CONFIG_PANIC_ON_OOPS"* ]; then continue; fi
      value="$(grep "^${line}$" /linux/.config || true)"

      # It's okay to for the merging script to have simply not listed values we
      # require to be unset.
      if echo "${line}" | grep "is not set" >/dev/null && [ "$value" = "" ]; then
          continue
      fi
      if [ "${value}" = "${line}" ]; then
          continue
      fi

      echo "$line set incorrectly" && false
    done < $1
}

for config in "$@"; do
  check_config "$config"
done
