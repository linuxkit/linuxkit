# Using the perf utility with LinuxKit

The `perf` utility is a standard Linux tool to access performance
counters, trace events and access various other kernel internals for
performance analysis.

The `perf` utility needs to matched be with the kernel. For recent
kernel build, LinuxKit provides a `linuxkit/kernel-perf` package with
a matching tag for each kernel under `linuxkit/kernel`.

There are a number of ways to use `linuxkit/kernel-perf` package:

1. Add it to the `init` section. This adds `/usr/bin/perf` to the
  systems' root filesystem. From there it can be
  - bind mounted into your container
  - accessed via `/proc/1/root/usr/bin/perf` from with in the `getty`
    or `ssh` container.
2. Add it to you package. If you have a custom package already, you
   can add `linuxkit/kernel-perf` as another stage in your package and
   then copy `/usr/bin/perf` into the final stage.

The first method is preferable since you need to match with the kernel
package tag and that is typically defined in the YAML file. I
typically don't add the bind mount since this requires further
modification and simply create a symlink in the `ssh` or `getty` container:

```
ln -s /proc/1/root/usr/bin/perf /usr/bin/perf
```

If you want to use `perf` you may also want to remove the `sysctl`
container, or alternatively, disable the kernel pointer restriction it
enables by default:

```
echo 0 > /proc/sys/kernel/kptr_restrict
```

Now, `perf` is ready to use. The LinuxKit `perf` package only contains
the `perf` binary, but excludes the detailed help messages or
additional scripts. If there is demand, we can add them to the
LinuxKit package.


