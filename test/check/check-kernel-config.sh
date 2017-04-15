#!/bin/sh

set -e

echo "starting kernel config sanity test with /proc/config.gz"

# decompress /proc/config.gz from the host
UNZIPPED_CONFIG=$(zcat /proc/config.gz)

kernelVersion="$(uname -r)"
kernelMajor="${kernelVersion%%.*}"
kernelMinor="${kernelVersion#$kernelMajor.}"
kernelMinor="${kernelMinor%%.*}"

# Most tests against https://kernsec.org/wiki/index.php/Kernel_Self_Protection_Project
# Positive cases

echo $UNZIPPED_CONFIG | grep -q CONFIG_BUG=y || (echo "CONFIG_BUG=y" && exit 1)
echo $UNZIPPED_CONFIG | grep -q CONFIG_DEBUG_KERNEL=y || (echo "CONFIG_DEBUG_KERNEL=y" && exit 1)
echo $UNZIPPED_CONFIG | grep -q CONFIG_DEBUG_RODATA=y || (echo "CONFIG_DEBUG_RODATA=y" && exit 1)
echo $UNZIPPED_CONFIG | grep -q CONFIG_CC_STACKPROTECTOR=y || (echo "CONFIG_CC_STACKPROTECTOR=y" && exit 1)
echo $UNZIPPED_CONFIG | grep -q CONFIG_CC_STACKPROTECTOR_STRONG=y || (echo "CONFIG_CC_STACKPROTECTOR_STRONG=y" && exit 1)
echo $UNZIPPED_CONFIG | grep -q CONFIG_STRICT_DEVMEM=y || (echo "CONFIG_STRICT_DEVMEM=y" && exit 1)
echo $UNZIPPED_CONFIG | grep -q CONFIG_SYN_COOKIES=y || (echo "CONFIG_SYN_COOKIES=y" && exit 1)
echo $UNZIPPED_CONFIG | grep -q CONFIG_DEBUG_CREDENTIALS=y || (echo "CONFIG_DEBUG_CREDENTIALS=y" && exit 1)
echo $UNZIPPED_CONFIG | grep -q CONFIG_DEBUG_NOTIFIERS=y || (echo "CONFIG_DEBUG_NOTIFIERS=y" && exit 1)
echo $UNZIPPED_CONFIG | grep -q CONFIG_DEBUG_LIST=y || (echo "CONFIG_DEBUG_LIST=y" && exit 1)
echo $UNZIPPED_CONFIG | grep -q CONFIG_SECCOMP=y || (echo "CONFIG_SECCOMP=y" && exit 1)
echo $UNZIPPED_CONFIG | grep -q CONFIG_SECCOMP_FILTER=y || (echo "CONFIG_SECCOMP_FILTER=y" && exit 1)
echo $UNZIPPED_CONFIG | grep -q CONFIG_SECURITY=y || (echo "CONFIG_SECURITY=y" && exit 1)
echo $UNZIPPED_CONFIG | grep -q CONFIG_SECURITY_YAMA=y || (echo "CONFIG_SECURITY_YAMA=y" && exit 1)
echo $UNZIPPED_CONFIG | grep -q CONFIG_PANIC_ON_OOPS=y || (echo "CONFIG_PANIC_ON_OOPS=y" && exit 1)
echo $UNZIPPED_CONFIG | grep -q CONFIG_DEBUG_SET_MODULE_RONX=y || (echo "CONFIG_DEBUG_SET_MODULE_RONX=y" && exit 1)
echo $UNZIPPED_CONFIG | grep -q CONFIG_SYN_COOKIES=y || (echo "CONFIG_SYN_COOKIES=y" && exit 1)
echo $UNZIPPED_CONFIG | grep -q CONFIG_LEGACY_VSYSCALL_NONE=y || (echo "CONFIG_LEGACY_VSYSCALL_NONE=y" && exit 1)
echo $UNZIPPED_CONFIG | grep -q CONFIG_RANDOMIZE_BASE=y || (echo "CONFIG_RANDOMIZE_BASE=y" && exit 1)

# Conditional on kernel version
if [ "$kernelMajor" -ge 4 -a "$kernelMinor" -ge 5 ]; then
  echo $UNZIPPED_CONFIG | grep -q CONFIG_IO_STRICT_DEVMEM=y || (echo "CONFIG_IO_STRICT_DEVMEM=y" && exit 1)
  echo $UNZIPPED_CONFIG | grep -q CONFIG_UBSAN=y || (echo "CONFIG_UBSAN=y" && exit 1)
fi
if [ "$kernelMajor" -ge 4 -a "$kernelMinor" -ge 7 ]; then
  echo $UNZIPPED_CONFIG | grep -q CONFIG_SLAB_FREELIST_RANDOM=y || (echo "CONFIG_SLAB_FREELIST_RANDOM=y" && exit 1)
fi
if [ "$kernelMajor" -ge 4 -a "$kernelMinor" -ge 8 ]; then
  echo $UNZIPPED_CONFIG | grep -q CONFIG_HARDENED_USERCOPY=y || (echo "CONFIG_HARDENED_USERCOPY=y" && exit 1)
  echo $UNZIPPED_CONFIG | grep -q CONFIG_RANDOMIZE_MEMORY=y || (echo "CONFIG_RANDOMIZE_MEMORY=y" && exit 1)
fi

# poisoning cannot be enabled in 4.4
if [ "$kernelMajor" -ge 4 -a "$kernelMinor" -ge 9 ]; then
  echo $UNZIPPED_CONFIG | grep -q CONFIG_PAGE_POISONING=y || (echo "CONFIG_PAGE_POISONING=y" && exit 1)
  echo $UNZIPPED_CONFIG | grep -q CONFIG_PAGE_POISONING_NO_SANITY=y || (echo "CONFIG_PAGE_POISONING_NO_SANITY=y" && exit 1)
  echo $UNZIPPED_CONFIG | grep -q CONFIG_PAGE_POISONING_ZERO=y || (echo "CONFIG_PAGE_POISONING_ZERO=y" && exit 1)
fi

if [ "$kernelMajor" -ge 4 -a "$kernelMinor" -ge 10 ]; then
  echo $UNZIPPED_CONFIG | grep -q CONFIG_BUG_ON_DATA_CORRUPTION=y || (echo "CONFIG_BUG_ON_DATA_CORRUPTION=y" && exit 1)
fi

# Negative cases
echo $UNZIPPED_CONFIG | grep -q 'CONFIG_ACPI_CUSTOM_METHOD is not set' || (echo "CONFIG_ACPI_CUSTOM_METHOD is not set" && exit 1)
echo $UNZIPPED_CONFIG | grep -q 'CONFIG_COMPAT_BRK is not set' || (echo "CONFIG_COMPAT_BRK is not set" && exit 1)
echo $UNZIPPED_CONFIG | grep -q 'CONFIG_DEVKMEM is not set' || (echo "CONFIG_DEVKMEM is not set" && exit 1)
echo $UNZIPPED_CONFIG | grep -q 'CONFIG_COMPAT_VDSO is not set' || (echo "CONFIG_COMPAT_VDSO is not set" && exit 1)
echo $UNZIPPED_CONFIG | grep -q 'CONFIG_KEXEC is not set' || (echo "CONFIG_KEXEC is not set" && exit 1)
echo $UNZIPPED_CONFIG | grep -q 'CONFIG_HIBERNATION is not set' || (echo "CONFIG_HIBERNATION is not set" && exit 1)
echo $UNZIPPED_CONFIG | grep -q 'CONFIG_LEGACY_PTYS is not set' || (echo "CONFIG_LEGACY_PTYS is not set" && exit 1)
echo $UNZIPPED_CONFIG | grep -q 'CONFIG_X86_X32 is not set' || (echo "CONFIG_X86_X32 is not set" && exit 1)
echo $UNZIPPED_CONFIG | grep -q 'CONFIG_MODIFY_LDT_SYSCALL is not set' || (echo "CONFIG_MODIFY_LDT_SYSCALL is not set" && exit 1)

echo "kernel config test succeeded!"
