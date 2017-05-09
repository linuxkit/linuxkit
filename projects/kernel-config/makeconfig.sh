#!/bin/bash

set -e

ARCH=$1
KERNEL_SERIES=$2
DEBUG=$3

defconfig=defconfig
if [ "${ARCH}" == "x86" ]; then
    defconfig=x86_64_defconfig
fi
configpath="/linux/arch/${ARCH}/configs/${defconfig}"

cp /config/kernel_config.base "$configpath"

function append_config()
{
    config=$1

    if [ -f "$config" ]; then
        cat "$config" >> "$configpath"
    fi
}

append_config "/config/kernel_config.${ARCH}"
append_config "/config/kernel_config.${KERNEL_SERIES}"
append_config "/config/kernel_config.${ARCH}.${KERNEL_SERIES}"

if [ -n "${DEBUG}" ]; then
    sed -i sed -i 's/CONFIG_PANIC_ON_OOPS=y/# CONFIG_PANIC_ON_OOPS is not set/' /linux/arch/x86/configs/x86_64_defconfig
    append_config "/config/kernel_config.debug"
fi

cd /linux && make defconfig && make oldconfig

# Let's make sure things are the way we want, i.e. every option we explicitly
# set is set the same way in the resulting config.
function check_config()
{
    if [ ! -f "$1" ]; then return; fi

    while read line; do
      if [ -n "${DEBUG}" ] && [ "$line" == "CONFIG_PANIC_ON_OOPS=y" ]; then continue; fi
      grep "^${line}$" /linux/.config >/dev/null || (echo "$line set incorrectly" && false)
    done < $1
}

check_config "/config/kernel_config.base"
check_config "/config/kernel_config.${ARCH}"
check_config "/config/kernel_config.${KERNEL_SERIES}"
check_config "/config/kernel_config.${ARCH}.${KERNEL_SERIES}"
if [ -n "${DEBUG}" ]; then
    check_config "/config/kernel_config.debug"
fi
