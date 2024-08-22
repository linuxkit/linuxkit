# Kernel command-line options

The kernel command-line is a string of text that the kernel parses as it is starting up. It is passed by the boot loader
to the kernel and specifies parameters that the kernel uses to configure the system. The command-line is a list of command-line
options separated by spaces. The options are parsed by the kernel and can be used to enable or disable certain features.

LinuxKit passes all command-line options to the kernel, which uses them in the usual way.

There are several options that can be used to control the behaviour of linuxkit itself, or specifically packages
within linuxkit. Unless standard Linux options exist, these all are prefaced with `linuxkit.`.

| Option | Description |
|---|---|
| `linuxkit.runc_debug=1` | Start runc for `onboot` and `onshutdown` containers to run with `--debug`. Also sends output to the console, in addition to the normal output to logfiles. If not present or set to 0, default to usual mode. |
| `linuxkit.unified_cgroup_hierarchy=0` | Start up cgroups v1. If not present or set to 1, default to cgroups v1. |
