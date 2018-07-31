# Using the bcc utility with LinuxKit

The `bcc` utility is a standard Linux tool to access performance
counters, trace events and access various other kernel internals for
performance analysis.

The `bcc` utility needs to matched be with the kernel. For recent
kernel build, LinuxKit provides a `linuxkit/kernel-bcc` package with
a matching tag for each kernel under `linuxkit/kernel`.

The preferred way of using the `linuxkit/kernel-bcc` package is to
add it to the `init` section. This adds `/usr/share/bcc` to the
  systems' root filesystem. From there it can be
  - bind mounted into your container
  - accessed via `/proc/1/root/usr/share/bcc/tools` from with in the `getty`
    or `ssh` container.
  - accessed via a nsenter of `/bin/ash` of proc 1.

If you want to use `bcc` you may also want to remove the `sysctl`
container, or alternatively, disable the kernel pointer restriction it
enables by default:

```
echo 0 > /proc/sys/kernel/kptr_restrict
```

Now, `bcc` is ready to use. The LinuxKit `bcc` package contains
the `bcc` binary, example and tool scripts, and kernel headers for the
associated kernel build.


