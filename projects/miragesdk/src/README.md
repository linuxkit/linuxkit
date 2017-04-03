### SDK

To build and test the SDK, run:

```
$ make test
```

This will work on any OS.

### DHCP client using MirageOS

To build the MirageOS DHCP client, run:

```
$ make dev
```

As this is using some BPF runes, this will work only on Linux. To debug/build
on OSX, you can create a container and build from there:

```
make enter-dev
# now in the dev container
make dev
```

### Documentation

See the [general architecture document](../../doc/unikernel.md).
