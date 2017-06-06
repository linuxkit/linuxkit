#### Vsudd

Vsudd is a daemon that forwards unix domain socket traffic from the host to the
guest VM via VSOCK. It can be used to control other daemons, like `containerd`
and `dockerd`, from the host.  An example configuration file is available in
[examples/vsudd.yml](/examples/vsudd.yml).

After building the example, run the example with `linuxkit run hyperkit
-vsock-ports 2374 vsudd`. This will create a unix domain socket in the state
directory that map to the `containerd` control socket. The socket is called
`guest.00000946`.

If you install the `ctr` tool on the host you should be able to access the
`containerd` running in the VM:

```
$ go get -u -ldflags -s github.com/containerd/containerd/cmd/ctr
...
$ ctr -a vsudd-state/guest.00000946 list
ID        IMAGE     PID       STATUS
vsudd               466       RUNNING
```

