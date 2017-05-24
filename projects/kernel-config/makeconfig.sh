#!/bin/bash

set -e

ARCH=$1
KERNEL_SERIES=$2
DEBUG=$3

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

merge_config "/config/kernel_config.base"
merge_config "/config/kernel_config.${ARCH}"
merge_config "/config/kernel_config.${KERNEL_SERIES}"
merge_config "/config/kernel_config.${ARCH}.${KERNEL_SERIES}"

if [ -n "${DEBUG}" ]; then
    sed -i sed -i 's/CONFIG_PANIC_ON_OOPS=y/# CONFIG_PANIC_ON_OOPS is not set/' /linux/arch/x86/configs/x86_64_defconfig
    append_config "/config/kernel_config.debug"
fi

cd /linux && make oldconfig

# Let's make sure things are the way we want, i.e. every option we explicitly
# set is set the same way in the resulting config.
function check_config()
{
    if [ ! -f "$1" ]; then return; fi

    while read line; do
      if [ -n "${DEBUG}" ] && [ "$line" == "CONFIG_PANIC_ON_OOPS=y" ]; then continue; fi
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

check_config "/config/kernel_config.base"
check_config "/config/kernel_config.${ARCH}"
check_config "/config/kernel_config.${KERNEL_SERIES}"
check_config "/config/kernel_config.${ARCH}.${KERNEL_SERIES}"
if [ -n "${DEBUG}" ]; then
    check_config "/config/kernel_config.debug"
fi
