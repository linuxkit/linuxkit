#!/bin/sh

set -e

function fail {
	printf "FAILURE: $1\n"
	FAILED=1
}

echo "starting kernel config sanity test with ${1:-/proc/config.gz}"

if [ -n "$1" ]; then
  UNZIPPED_CONFIG=$(cat "$1")
else
  # decompress /proc/config.gz from the host
  UNZIPPED_CONFIG=$(zcat /proc/config.gz)
fi

kernelVersion="$(uname -r)"
kernelMajor="${kernelVersion%%.*}"
kernelMinor="${kernelVersion#$kernelMajor.}"
kernelMinor="${kernelMinor%%.*}"

# Most tests against https://kernsec.org/wiki/index.php/Kernel_Self_Protection_Project
# Positive cases

echo $UNZIPPED_CONFIG | grep -q CONFIG_BUG=y || fail "CONFIG_BUG=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_DEBUG_KERNEL=y || fail "CONFIG_DEBUG_KERNEL=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_CC_STACKPROTECTOR=y || fail "CONFIG_CC_STACKPROTECTOR=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_CC_STACKPROTECTOR_STRONG=y || fail "CONFIG_CC_STACKPROTECTOR_STRONG=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_STRICT_DEVMEM=y || fail "CONFIG_STRICT_DEVMEM=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_SYN_COOKIES=y || fail "CONFIG_SYN_COOKIES=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_DEBUG_CREDENTIALS=y || fail "CONFIG_DEBUG_CREDENTIALS=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_DEBUG_NOTIFIERS=y || fail "CONFIG_DEBUG_NOTIFIERS=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_DEBUG_LIST=y || fail "CONFIG_DEBUG_LIST=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_SECCOMP=y || fail "CONFIG_SECCOMP=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_SECCOMP_FILTER=y || fail "CONFIG_SECCOMP_FILTER=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_SECURITY=y || fail "CONFIG_SECURITY=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_SECURITY_YAMA=y || fail "CONFIG_SECURITY_YAMA=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_PANIC_ON_OOPS=y || fail "CONFIG_PANIC_ON_OOPS=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_SYN_COOKIES=y || fail "CONFIG_SYN_COOKIES=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_LEGACY_VSYSCALL_NONE=y || fail "CONFIG_LEGACY_VSYSCALL_NONE=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_RANDOMIZE_BASE=y || fail "CONFIG_RANDOMIZE_BASE=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_PAGE_TABLE_ISOLATION=y || fail "CONFIG_PAGE_TABLE_ISOLATION=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_RETPOLINE=y || fail "CONFIG_RETPOLINE=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_GENERIC_CPU_VULNERABILITIES=y || fail "CONFIG_GENERIC_CPU_VULNERABILITIES=y"
echo $UNZIPPED_CONFIG | grep -q CONFIG_BPF_JIT_ALWAYS_ON=y || fail "CONFIG_BPF_JIT_ALWAYS_ON=y"


# Conditional on kernel version
if [ "$kernelMajor" -ge 4 -a "$kernelMinor" -ge 5 ]; then
  echo $UNZIPPED_CONFIG | grep -q CONFIG_IO_STRICT_DEVMEM=y || fail "CONFIG_IO_STRICT_DEVMEM=y"
  echo $UNZIPPED_CONFIG | grep -q CONFIG_UBSAN=y || fail "CONFIG_UBSAN=y"
fi
if [ "$kernelMajor" -ge 4 -a "$kernelMinor" -ge 7 ]; then
  echo $UNZIPPED_CONFIG | grep -q CONFIG_SLAB_FREELIST_RANDOM=y || fail "CONFIG_SLAB_FREELIST_RANDOM=y"
fi
if [ "$kernelMajor" -ge 4 -a "$kernelMinor" -ge 8 ]; then
  echo $UNZIPPED_CONFIG | grep -q CONFIG_HARDENED_USERCOPY=y || fail "CONFIG_HARDENED_USERCOPY=y"
  echo $UNZIPPED_CONFIG | grep -q CONFIG_RANDOMIZE_MEMORY=y || fail "CONFIG_RANDOMIZE_MEMORY=y"
fi

# poisoning cannot be enabled in 4.4
if [ "$kernelMajor" -ge 4 -a "$kernelMinor" -ge 9 ]; then
  echo $UNZIPPED_CONFIG | grep -q CONFIG_PAGE_POISONING=y || fail "CONFIG_PAGE_POISONING=y"
  echo $UNZIPPED_CONFIG | grep -q CONFIG_PAGE_POISONING_NO_SANITY=y || fail "CONFIG_PAGE_POISONING_NO_SANITY=y"
  echo $UNZIPPED_CONFIG | grep -q CONFIG_PAGE_POISONING_ZERO=y || fail "CONFIG_PAGE_POISONING_ZERO=y"
fi

if [ "$kernelMajor" -ge 4 -a "$kernelMinor" -ge 10 ]; then
  echo $UNZIPPED_CONFIG | grep -q CONFIG_BUG_ON_DATA_CORRUPTION=y || fail "CONFIG_BUG_ON_DATA_CORRUPTION=y"
fi

if [ "$kernelMajor" -ge 4 -a "$kernelMinor" -le 10 ]; then
  echo $UNZIPPED_CONFIG | grep -q CONFIG_DEBUG_RODATA=y || fail "CONFIG_DEBUG_RODATA=y"
  echo $UNZIPPED_CONFIG | grep -q CONFIG_DEBUG_SET_MODULE_RONX=y || fail "CONFIG_DEBUG_SET_MODULE_RONX=y"
fi

if [ "$kernelMajor" -ge 4 -a "$kernelMinor" -ge 11 ]; then
  echo $UNZIPPED_CONFIG | grep -q CONFIG_STRICT_KERNEL_RWX=y || fail "CONFIG_STRICT_KERNEL_RWX=y"
  echo $UNZIPPED_CONFIG | grep -q CONFIG_STRICT_MODULE_RWX=y || fail "CONFIG_STRICT_MODULE_RWX=y"
fi

# Negative cases
echo $UNZIPPED_CONFIG | grep -q 'CONFIG_ACPI_CUSTOM_METHOD is not set' || fail "CONFIG_ACPI_CUSTOM_METHOD is not set"
echo $UNZIPPED_CONFIG | grep -q 'CONFIG_COMPAT_BRK is not set' || fail "CONFIG_COMPAT_BRK is not set"
echo $UNZIPPED_CONFIG | grep -q 'CONFIG_DEVKMEM is not set' || fail "CONFIG_DEVKMEM is not set"
echo $UNZIPPED_CONFIG | grep -q 'CONFIG_COMPAT_VDSO is not set' || fail "CONFIG_COMPAT_VDSO is not set"
echo $UNZIPPED_CONFIG | grep -q 'CONFIG_KEXEC is not set' || fail "CONFIG_KEXEC is not set"
echo $UNZIPPED_CONFIG | grep -q 'CONFIG_HIBERNATION is not set' || fail "CONFIG_HIBERNATION is not set"
echo $UNZIPPED_CONFIG | grep -q 'CONFIG_LEGACY_PTYS is not set' || fail "CONFIG_LEGACY_PTYS is not set"
echo $UNZIPPED_CONFIG | grep -q 'CONFIG_X86_X32 is not set' || fail "CONFIG_X86_X32 is not set"
echo $UNZIPPED_CONFIG | grep -q 'CONFIG_MODIFY_LDT_SYSCALL is not set' || fail "CONFIG_MODIFY_LDT_SYSCALL is not set"
echo $UNZIPPED_CONFIG | grep -q 'CONFIG_SCSI_PROC_FS is not set' || fail "CONFIG_SCSI_PROC_FS is not set"

# modprobe
for mod in \
nfs \
nfsd \
ntfs
do
	modprobe $mod 2>/dev/null || true
done

# check filesystems that are built in
for fs in \
sysfs \
rootfs \
tmpfs \
bdev \
proc \
cpuset \
cgroup \
devtmpfs \
binfmt_misc \
debugfs \
tracefs \
securityfs \
sockfs \
bpf \
pipefs \
ramfs \
hugetlbfs \
rpc_pipefs \
devpts \
ext4 \
vfat \
msdos \
iso9660 \
nfs \
nfs4 \
nfsd \
cifs \
ntfs \
fuseblk \
fuse \
fusectl \
overlay \
udf \
xfs \
9p \
pstore \
mqueue \
oprofilefs
do
	grep -q "[[:space:]]${fs}\$" /proc/filesystems || fail "${fs} filesystem missing"
done

if [ -z "$FAILED" ]
then
	echo "kernel config test succeeded!"
else
	echo "kernel config test failed!"
	exit 1
fi
