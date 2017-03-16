## Roadmap

Very basic roadmap, to be improved shortly.

### Done

- use 2 static binaries privileged + unikernel (calf) in the container,
  connected via socketpairs and pipes.
- use eBPF to filter DHCP traffic
- redirect the calf's sterr/stdout to the priv container
- the priv exposes a simple HTTP interface to the calf, and read/write
  are stored into a local Datakit/Git repo.
- use upstream MirageOS's charrua-core.

### TODO

- current: make the packets flow in both directions
- use runc to isolate the calf
- use seccomp to isolate the privileged container
- use the DHCP results to actually update the system
- add metrics aggregation (using prometheus)
- better logging aggregation (using syslog)
