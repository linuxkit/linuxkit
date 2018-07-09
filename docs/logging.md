# Logging

By default LinuxKit will write onboot and service logs directly to files in
`/var/log` and `/var/log/onboot`.

It is tricky to write the logs to a disk or a network service as no disks
or networks are available until the `onboot` containers run. We work around
this by splitting the logging into 2 pieces:

1. `memlogd`: an in-memory circular buffer which receives logs (including
   all the early `onboot` logs)
2. a log writing `service` that starts later and can download and process
   the logs from `memlogd`

To use this new logging system, you should add the `memlogd` container to
the `init` block in the LinuxKit yml. On boot `memlogd` will be started
from `init.d` and it will listen on a Unix domain socket:

```
/var/run/linuxkit-external-logging.sock
```

The `init`/`service` process will look for this socket and redirect the
`stdout` and `stderr` of both `onboot` and `services` to `memlogd`.

## memlogd: an in-memory circular buffer

The `memlogd` daemon reads the logs from the `onboot` and `services` containers
and stores them together with a timestamp and the name of the originating
container in a circular buffer in memory.

The contents of the circular buffer can be read over the Unix domain socket
```
/var/run/memlogq.sock
```

The circular buffer has a fixed size (overridden by the command-line argument
`-max-lines`) and when it fills up, the oldest messages will be overwritten.

To store the logs somewhere more permanent, for example a disk or a remote
network service, a service should be added to the yaml which connects to
`memlogd` and streams the logs. The `logwrite` service described below shows
how to do this.

### Message format

The format used to read logs is similar to [kmsg](https://www.kernel.org/doc/Documentation/ABI/testing/dev-kmsg):
```
<timestamp>,<log>;<body>
```
where `<timestamp>` is an RFC3339-formatted timestamp, `<log>` is the name of
the log (e.g. `docker-ce.out`) and `<body>` is the output. The `<log>` must
not contain the character `;`.

## logwrite: writing logs to disk

The service `pkg/logwrite` connects to `memlogd` and streams the logs to files
in `/var/log`. The logs are automatically rotated; by default each file has
a maximum size of 1 MiB and up to 10 files are kept per log. The arguments
`-max-log-files` and `-max-log-size` can be used to override these defaults.

Here is an example log file:
```
# cat /var/log/onboot.001-dhcpcd.out 
2018-07-08T09:16:53Z onboot.001-dhcpcd.out eth0: waiting for carrier
2018-07-08T09:16:53Z onboot.001-dhcpcd.out eth0: carrier acquired
2018-07-08T09:16:53Z onboot.001-dhcpcd.out DUID 00:01:00:01:22:d4:93:05:02:50:00
:00:00:06
2018-07-08T09:16:53Z onboot.001-dhcpcd.out eth0: IAID 00:00:00:06
2018-07-08T09:16:53Z onboot.001-dhcpcd.out eth0: adding address fe80::f346:56a6:590d:5ea4
2018-07-08T09:16:53Z onboot.001-dhcpcd.out eth0: soliciting an IPv6 router
2018-07-08T09:16:53Z onboot.001-dhcpcd.out eth0: soliciting a DHCP lease
2018-07-08T09:16:53Z onboot.001-dhcpcd.out eth0: offered 192.168.65.8 from 192.168.65.1 `vpnkit'
2018-07-08T09:16:53Z onboot.001-dhcpcd.out eth0: leased 192.168.65.8 for 7200 se
conds
2018-07-08T09:16:53Z onboot.001-dhcpcd.out eth0: adding route to 192.168.65.0/24
2018-07-08T09:16:53Z onboot.001-dhcpcd.out eth0: adding default route via 192.16
8.65.1
2018-07-08T09:16:53Z onboot.001-dhcpcd.out exiting due to oneshot
2018-07-08T09:16:53Z onboot.001-dhcpcd.out dhcpcd exited
```

## Current issues and limitations:

- No docker logger plugin support yet - it could be nice to add support to
  memlogd, so the docker container logs would also be gathered in one place
- No syslog compatibility at the moment and `/dev/log` doesn’t exist. This
  socket could be created to keep syslog compatibility, e.g. by using
  https://github.com/mcuadros/go-syslog. Processes that require syslog should
  then be able to log directly to memlogd.
- Currently no direct external hooks exposed - but options available that
  could be added. Should also be possible to pipe output to e.g. `oklog`
  from `logread` (https://github.com/oklog/oklog)

