## Unikernel System Containers

### General Architecture

```
               |=================|                      |================|
               | privileged shim |                      |       calf     |
               |=================|                      |================|
               |                 |                      |                |
<--  eth0 ---> |    eBPF rules   | <--- network IO ---> |   type-safe    |
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
   diagnostic  |     deamon      |     (control path)   |     client     |
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

1. privileged shim (privileged system service)
  - run in a privileged container
  - can read all network traffic
  - can set-up eBPF rules (or a dumb forwarder to start with)
  - exposes an easily auditable KV store for configuration values
    (over a simple REST/HTTP API to start with).
    Expose a scoped "view" of the config store to the
    calf (a different branch in a datakit store for instance) and another
    unscoped "view" to the host (could be a Git log if using datakit).
  - has a set of system handlers who watches for changes in the KV
    store and perform privileged operations inside moby (syscalls, edit
    global config files, etc). System handlers use the config store CLI
    to wait for events and react.

2. calf (sandboxed system service)
  - run in a fully isolated container
  - full sandbox (initially a normal Unix process, later on unielf/wasm)
  - has a type-safe network stack to handle network IO
  - has type-safe business logic to process network IO
  - has a limited access read and write access to the config store where the
    result of the business logic is output

### DHCP client

#### Shim

- forward DHCP traffic only (in both directions)
- expose a simple store to the calf, with the following keys:

```
/ip (mandatory)
/mtu (optional)
/domain (optional)
/search (optional)
/nameserver/001 (optional)
...
/nameserver/xxx (optional)
```

If runs a small webserver where it exposes a simple CRUD interface
over these keys -- only the calf can see it (e.g. it opens a pipe and
share it with the calf on startup).

- system handlers:
  - if /ip change -> set IP address on moby host
  - if /domain change -> set moby domain name
  - if /search -> set search domain on moby host
  - if /nameserver/xxx -> set DNS servers on moby

#### Calf

- MirageOS unikernel using charrua-client (or a fork of it).
- Has access to a Mirage_net.S interface for network traffic
- Has access to a a simple KV interface

Internally, it uses something more typed than a KV store:

```
module Shim: sig
  val set_ip: Ipaddr.V4.t -> unit Lwt.t
  val set_domain: string -> unit Lwt.t
  val set_search: string -> unit Lwt.t
  val set_nameservers: Ipaddr.V4.t list Lwt.t
end
```

but this ends up being translated into REST/RPC calls to the shim.

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

### Roadmap

#### first PoC: DHCP client

Current status: one container containing two static binaries (priv + calf),
private pipes open between the process for stdout/stderr aggregation +
raw sockets (data path). Control path is using a simple HTTP server running
in the priv container. The calf is using the dev version of mirage/charrua-core,
and is able to get a DHCP lease on boot.

##### TODO

- use runc to isolate the calf
- system handler (see https://github.com/kobolabs/dhcpcd/tree/kobo/dhcpcd-hooks)
- use seccomp to isolate the privileged container
- use the DHCP results to actually update the system
- add metrics aggregation (using prometheus)
- better logging aggregation (using syslog)
- IPv6 support
- tests, tests, tests (especially against non compliant RFC servers)

### Second iteration: NTP

TODO
