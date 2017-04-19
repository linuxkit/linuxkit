## Unikernel System Containers

### General Architecture

```
               |=================|                      |================|
               |       priv      |                      |       calf     |
               |=================|                      |================|
               |                 |                      |                |
<--  eth0 ---> |    BPF rules    | <--- network IO ---> |   type-safe    |
               |                 |      (data path)     | network stack  |
               |                 |                      |                |
               |-----------------|                      |----------------|
               |                 |                      |                |
<-- logs ----- |                 | <------- logs ------ |   type-safe    |
               |                 |                      | protocol logic |
<-- metrics -- |                 | <----- metrics ----- |                |
               |                 |                      |                |
               |-----------------|                      |----------------|
               |                 |                      |                |
<-- audit ---  |  config store   | <----- KV store ---> |  config store  |
   diagnostic  |     daemon      |     (control path)   |     client     |
               |                 |                      |                |
               |_________________|                      |________________|
               |                 |
<-- sycalls -- |                 |
               |                 |
               | system handlers |
<-- config --- |                 |
    files      |                 |
               |_________________|
```

#### Priv: privileged system service

- run in a privileged container (but can have limited capabilities + seccomp)
- can read all network traffic
- can set-up (e)BPF rules
- exposes an easily auditable KV store for configuration values
- has a set of system handlers who watches for changes in the KV
    store and perform privileged operations inside moby (syscalls, edit
    of global config files, etc)

#### Calf: sandboxed system service

- run in a fully isolated container
- full sandbox (initially a normal Unix process, later on unielf/wasm)
- has a type-safe network stack to handle network IO
- has type-safe business logic to process network IO
- has a limited access read and write access to the config store where the
  result of the business logic is output

### DHCP client

#### Priv

- The privileged system service forwards DHCP traffic in both directions and
  block all other traffic. This is ensured by setting up BPF filters on the
  network interface.

- The privileged system service initialize the calf by opening the file
  descriptors for the control and data paths and calling `runc`.

- The privileged system service exposes a simple KV store to the calf, using
  the following keys:

    ```
    # read-only, set on startup by the priv
    /mac

    # write-only, set by the calf when it gots a lease
    /ip
    /gateway
    /mtu
    /domain
    /search
    /nameserver/001
    ...
    /nameserver/xxx
    ```

  The the KV store API is defined in term of [cap-n-proto](https://capnproto.org/)
  prototype:

    ```capnp
    @0x9e83562906de8259;

    struct Request {
      id   @0 :Int32;
      path @1 :List(Text);
      union {
        write  @2 :Data;
        read   @3 :Void;
        delete @4 :Void;
      }
    }

    struct Response {
      id   @0: Int32;
      union {
        ok    @1 :Data;
        error @2 :Data;
      }
    }
    ```

- The privileged system service installs the following system handlers:
  - if /ip change -> bring up the default interface and set IP address (done)
  - if /gateway change -> set up route (done)
  - if /domain change -> set moby domain name (todo)
  - if /search -> set search domain on moby host (todo)
  - if /nameserver/xxx -> set DNS servers on moby (todo)

- The privileged system service updates configuration files:
  - /ect/resolv.conf (todo)

#### Calf

- The sandboxed system service is a MirageOS unikernel using [charrua-core](https://github.com/mirage/charrua-core).
- The sandboxed system service reads the DHCP network traffic from an already
  opened file descriptor.
- The sandboxed system service reads and sets the control state using and
  already opened file descriptor,

### SDK

What the SDK should enable:
1. easily write a new calfs initially in OCaml, then Rust.
   Probably not very useful on its own.
2. easily write a new shim by providing the basic blocks:
   eBPF scripts, calf runner, KV store, system handlers.
   Initially could be a standalone blob, but should aim for
   independant and re-usable pieces that could run in a
   container.
3. (later) generate shim/caft containers from a single (API?)
   description.

See `./src/sdk` for the current state of the SDK.

### Roadmap

#### first PoC: DHCP client

##### TODO

- better system handler using language bindings instead of shelling out to ifconfig
- use seccomp to isolate the privileged container
- use mtu, domain, nameservers parameters
- generate resolv.conf
- add metrics aggregation (using prometheus)
- better logging aggregation (using syslog)
- IPv6 support
- tests, tests, tests (especially against non compliant RFC servers)

### Second iteration: NTP

TODO
