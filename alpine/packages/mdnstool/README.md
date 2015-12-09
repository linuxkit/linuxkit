Tool to monitor a network interface for IP changes and publish an mDNS service.

To publish `docker.local` and map it to the IP of interface `eth0`:

```
./mdnstool -if eth0
```

Options:

```
Usage of ./mdnstool:
  -hostname string
        Hostname - must be FQDN and end with . (default "docker.local.")
  -if string
        Network interface to bind multicast listener to. This interface will be monitored for IP changes. (default "eth0")
  -info string
        TXT service description (default "Moby")
  -instance string
        Instance description (default "Moby")
  -port int
        Service port (default 22)
  -service string
        SRV service type (default "_ssh._tcp")
```

To build for Linux:

```
GOOS=linux GOARCH=386 go build -v
```
