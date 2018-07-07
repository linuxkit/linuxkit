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
`memlogd` and streams the logs. The example program `logread` in the `memlogd`
package demonstrates how to do this.

### Message format

The format used to read logs is similar to [kmsg](https://www.kernel.org/doc/Documentation/ABI/testing/dev-kmsg):
```
<timestamp>,<log>;<body>
```
where `<timestamp>` is an RFC3339-formatted timestamp, `<log>` is the name of
the log (e.g. `docker-ce.out`) and `<body>` is the output. The `<log>` must
not contain the character `;`.

### Usage examples
```
/ # logread -f
2018-07-05T13:22:32Z,memlogd;memlogd started
2018-07-05T13:22:32Z,onboot.001-dhcpcd.out;eth0: waiting for carrier
2018-07-05T13:22:32Z,onboot.001-dhcpcd.err;eth0: could not detect a useable init
 system
2018-07-05T13:22:32Z,onboot.001-dhcpcd.out;eth0: carrier acquired
2018-07-05T13:22:32Z,onboot.001-dhcpcd.out;DUID 00:01:00:01:22:d0:d8:18:02:50:00:00:00:02
2018-07-05T13:22:32Z,onboot.001-dhcpcd.out;eth0: IAID 00:00:00:02
2018-07-05T13:22:32Z,onboot.001-dhcpcd.out;eth0: adding address fe80::d33a:3936:
2ee4:5c8c
2018-07-05T13:22:32Z,onboot.001-dhcpcd.out;eth0: soliciting an IPv6 router
2018-07-05T13:22:32Z,onboot.001-dhcpcd.out;eth0: soliciting a DHCP lease
2018-07-05T13:22:32Z,onboot.001-dhcpcd.out;eth0: offered 192.168.65.4 from 192.1
68.65.1 `vpnkit'
2018-07-05T13:22:32Z,onboot.001-dhcpcd.out;eth0: leased 192.168.65.4 for 7200 se
conds
2018-07-05T13:22:32Z,onboot.001-dhcpcd.out;eth0: adding route to 192.168.65.0/24
2018-07-05T13:22:32Z,onboot.001-dhcpcd.out;eth0: adding default route via 192.16
8.65.1
2018-07-05T13:22:32Z,onboot.001-dhcpcd.out;exiting due to oneshot
2018-07-05T13:22:32Z,onboot.001-dhcpcd.out;dhcpcd exited
^C
```

Current issues and limitations:

- No docker logger plugin support yet - it could be nice to add support to
  memlogd, so the docker container logs would also be gathered in one place
- No syslog compatibility at the moment and `/dev/log` doesn’t exist. This
  socket could be created to keep syslog compatibility, e.g. by using
  https://github.com/mcuadros/go-syslog. Processes that require syslog should
  then be able to log directly to memlogd.
- Kernel messages not read on startup yet (but can be captured with
  `logwrite dmesg`)
- Currently no direct external hooks exposed - but options available that
  could be added. Should also be possible to pipe output to e.g. `oklog`
  from `logread` (https://github.com/oklog/oklog)

