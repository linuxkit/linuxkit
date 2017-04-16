### Logging tools

Experimental logging tools for linuxkit. 

This project currently provides three tools for system logs; `logwrite`, `logread` and `memlogd` (+ `startmemlogd` to run `memlogd` with `runc`).

`memlogd` is the daemon that keeps logs in a circular buffer in memory. It is started automatically by `init`/`startmemlogd` in a runc container. It is passed two sockets - one that allows clients to dump/follow the logs and one that can be used to send open file descriptors to `memlogd`. When `memlogd` receives a file descriptor it will read from the file descriptor and timestamp and append the content to the in-memory log until the file is closed.

`logwrite` executes a command and will send stderr and stdout to `memlogd`. It does this by opening a socketpair for stdout and stderr and then sends the file descriptors to memlogd, before executing a specified command. Output is also sent to normal stderr/stdin. For example, `logwrite ls` will show the output both in the console and record it in the logs.

`logread` connects to memlogd and dumps the ring buffer. Parameters `-f` and `-F` can be used to follow the logs and disable the initial log dump (it behaves similar to busybox’ `logread`)

Init is modified to run all `onboot` and `service` containers wrapped in`logwrite` and to run `/usr/bin/startmemlogd`.

New sockets:
`/tmp/memlogd.sock` — sock_dgram which accepts an fd and a null-terminated source description
`/tmp/memlogdq.sock` — sock_stream to ask to dump/follow logs

Usage examples:
```
/ # logread -f
2017-04-15T15:37:37Z memlogd memlogd started
2017-04-15T15:37:37Z 002-dhcpcd.stdout eth0: waiting for carrier
2017-04-15T15:37:37Z 002-dhcpcd.stdout eth0: carrier acquired
2017-04-15T15:37:37Z 002-dhcpcd.stdout DUID 00:01:00:01:20:84:fa:c1:02:50:00:00:00:24
2017-04-15T15:37:37Z 002-dhcpcd.stdout eth0: IAID 00:00:00:24
2017-04-15T15:37:37Z 002-dhcpcd.stdout eth0: adding address fe80::84e3:ca52:2590:fe80
2017-04-15T15:37:37Z 002-dhcpcd.stdout eth0: soliciting an IPv6 router
2017-04-15T15:37:37Z 002-dhcpcd.stdout eth0: soliciting a DHCP lease
2017-04-15T15:37:37Z 002-dhcpcd.stdout eth0: offered 192.168.65.37 from 192.168.65.1 `vpnkit'
2017-04-15T15:37:37Z 002-dhcpcd.stdout eth0: leased 192.168.65.37 for 7199 seconds
2017-04-15T15:37:37Z 002-dhcpcd.stdout eth0: adding route to 192.168.65.0/24
2017-04-15T15:37:37Z 002-dhcpcd.stdout eth0: adding default route via 192.168.65.1
2017-04-15T15:37:37Z 002-dhcpcd.stdout exiting due to oneshot
2017-04-15T15:37:37Z 002-dhcpcd.stdout dhcpcd exited
2017-04-15T15:37:37Z rngd.stderr Unable to open file: /dev/tpm0
^C
/ # logwrite echo testing123
testing123
/ # logread | tail -n1
2017-04-15T15:37:45Z echo.stdout testing123
/ # echo -en "GET / HTTP/1.0\n\n" | nc localhost 80 > /dev/null
/ # logread | grep nginx
2017-04-15T15:42:40Z nginx.stdout 127.0.0.1 - - [15/Apr/2017:15:42:40 +0000] "GET / HTTP/1.0" 200 612 "-" "-" "-"
```

Current issues and limitations:

- The moby tool only supports onboot and service containers. `memlogd` runs as a special container that is managed by init, as it needs fd’s created in advance. To work around this a memlogd container is exported during build. The init-section in the yml is used to extract it to `/containers/init/memlogd` with a pre-created `config.json`.
- No docker logger plugin support yet - it could be nice to add support to memlogd, so the docker container logs would also be gathered in one place
- No syslog compatibility at the moment and `/dev/log` doesn’t exist. This socket could be created to keep syslog compatibility, e.g. by using https://github.com/mcuadros/go-syslog. Processes that require syslog should then be able to log directly to memlogd.
- Kernel messages not read on startup yet (but can be captured with `logwrite dmesg`)
- Currently no direct external hooks exposed - but options available that could be added. Should also be possible to pipe output to e.g. `oklog` from `logread` (https://github.com/oklog/oklog)

