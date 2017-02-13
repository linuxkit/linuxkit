#!/bin/sh

set -e

echo "starting kernel config sanity test with /proc/config.gz"

# decompress /proc/config.gz from the Moby host
zcat /proc/config.gz > unzipped_config

kernelVersion="$(uname -r)"
kernelMajor="${kernelVersion%%.*}"
kernelMinor="${kernelVersion#$kernelMajor.}"
kernelMinor="${kernelMinor%%.*}"

# Most tests against https://kernsec.org/wiki/index.php/Kernel_Self_Protection_Project
# Positive cases
cat unzipped_config | grep CONFIG_BUG=y
cat unzipped_config | grep CONFIG_DEBUG_KERNEL=y
cat unzipped_config | grep CONFIG_DEBUG_RODATA=y
cat unzipped_config | grep CONFIG_CC_STACKPROTECTOR=y
cat unzipped_config | grep CONFIG_CC_STACKPROTECTOR_STRONG=y
cat unzipped_config | grep CONFIG_STRICT_DEVMEM=y
cat unzipped_config | grep CONFIG_IO_STRICT_DEVMEM=y
cat unzipped_config | grep CONFIG_SYN_COOKIES=y
cat unzipped_config | grep CONFIG_DEBUG_CREDENTIALS=y
cat unzipped_config | grep CONFIG_DEBUG_NOTIFIERS=y
cat unzipped_config | grep CONFIG_DEBUG_LIST=y
cat unzipped_config | grep CONFIG_SECCOMP=y
cat unzipped_config | grep CONFIG_SECCOMP_FILTER=y
cat unzipped_config | grep CONFIG_SECURITY=y
cat unzipped_config | grep CONFIG_SECURITY_YAMA=y
cat unzipped_config | grep CONFIG_PANIC_ON_OOPS=y
cat unzipped_config | grep CONFIG_DEBUG_SET_MODULE_RONX=y
cat unzipped_config | grep CONFIG_HARDENED_USERCOPY=y
cat unzipped_config | grep CONFIG_SYN_COOKIES=y
cat unzipped_config | grep CONFIG_PAGE_POISONING=y
cat unzipped_config | grep CONFIG_PAGE_POISONING_NO_SANITY=y
cat unzipped_config | grep CONFIG_PAGE_POISONING_ZERO=y
cat unzipped_config | grep CONFIG_LEGACY_VSYSCALL_NONE=y
cat unzipped_config | grep CONFIG_BUG_ON_DATA_CORRUPTION=y
cat unzipped_config | grep CONFIG_RANDOMIZE_BASE=y
cat unzipped_config | grep CONFIG_RANDOMIZE_MEMORY=y

# Conditional on kernel version
if [ "$kernelMajor" -ge 4 -a "$kernelMinor" -ge 5 ]; then
  cat unzipped_config | grep CONFIG_IO_STRICT_DEVMEM=y
  cat unzipped_config | grep CONFIG_UBSAN=y
fi
if [ "$kernelMajor" -ge 4 -a "$kernelMinor" -ge 7 ]; then
  cat unzipped_config | grep CONFIG_SLAB_FREELIST_RANDOM=y
fi
if [ "$kernelMajor" -ge 4 -a "$kernelMinor" -ge 8 ]; then
  cat unzipped_config | grep CONFIG_HARDENED_USERCOPY=y
fi

# Negative cases
cat unzipped_config | grep 'CONFIG_ACPI_CUSTOM_METHOD is not set'
cat unzipped_config | grep 'CONFIG_COMPAT_BRK is not set'
cat unzipped_config | grep 'CONFIG_DEVKMEM is not set'
cat unzipped_config | grep 'CONFIG_COMPAT_VDSO is not set'
cat unzipped_config | grep 'CONFIG_KEXEC is not set'
cat unzipped_config | grep 'CONFIG_HIBERNATION is not set'
cat unzipped_config | grep 'CONFIG_LEGACY_PTYS is not set'
cat unzipped_config | grep 'CONFIG_X86_X32 is not set'
cat unzipped_config | grep 'CONFIG_MODIFY_LDT_SYSCALL is not set'
cat unzipped_config | grep 'CONFIG_KEXEC is not set'
cat unzipped_config | grep 'CONFIG_HIBERNATION is not set'
