## Using eBPF programs

There is now a development image `mobylinux/ebpf:_tag_`. These are currently being built
manually, I will tag one for each kernel release, as you should have a close one, eg
`mobylinux/ebpf:4.9` is currently available.

This image has all the kernel headers, `iovisor/bcc` built with support for C, Python and Lua,
and all sources installed. It is very large so if we are shipping stuff based on this we will
just extract compiled eBPF programs probably, but it is also usable for experiments, debug,
benchmarks etc.

You probably want to run with

`docker run -it -v /sys/kernel/debug:/sys/kernel/debug --privileged --pid=host mobylinux/ebpf:tag sh` for
interactive use as some things use debugfs. You need at least `CAP_SYS_ADMIN` to do anything.
There are examples in `bcc/examples` that should generally just work, I have tried several of
the Lua ones.

Some of the `iovisor/bcc` samples try to access the kernel symbols. For them to work correctly you should also execute:
```sh
echo 0 > /proc/sys/kernel/kptr_restrict
```
