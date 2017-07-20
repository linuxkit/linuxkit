# WireGuard

[WireGuard](https://www.wireguard.com) is a modern VPN released for the Linux kernel that can replace IPSec.

We can use WireGuard in Moby to better secure container networking.
WireGuard transparently encrypts *and* authenticates traffic between all peers, and uses state-of-the-art cryptography
from the [Noise protocol](https://noiseprotocol.org/). Moreover, WireGuard is implemented in less than a few thousand
lines of code, making it auditable for security.

Moreover, WireGuard provides a `wg0` (`wg1`, `wg2`,... etc) network interface that can be passed directly to containers,
such that all intercontainer traffic would benefit from encrypted and authenticated networking.

A full technical paper from NDSS 2017 is available [here](https://www.wireguard.com/papers/wireguard.pdf). The protocol has been formally verified, with a paper describing the security proofs available [here](https://www.wireguard.com/papers/wireguard-formal-verification.pdf).

## Contents

### Kernel Patches
The default kernels build WireGuard in as a module.

### Userspace Tools
The userspace tools are now a package available in `tools/alpine`, which can be installed via `apk add wireguard-tools`.

## Quickstart
To give WireGuard a spin, the [official quick start](https://www.wireguard.com/quickstart/) is a good way to get going.  For containers,
WireGuard has a [network namespace integration](https://www.wireguard.com/netns/) that we could use for Moby's containers.

## Roadmap

- We have yet to determine the best way to integrate WireGuard into Moby - at the node level or service level isolation.
  - Node level: it's plausible that Moby's provisioner could allocate keys per Moby node
  - Service level: swarmkit could set up WireGuard on a per-service basis, handing the container the wireguard interface

*Service Level*: one proposal is to use WireGuard between container network [`links`](https://docs.docker.com/compose/networking/#links).
This is a natural fit because WireGuard associates public keys to IP addresses: a docker-compose link would simply need
a reference to a key in addition to the existing IP address info for this to work.  However there are some open questions:
  - `containerd` does not intend to support networks from the roadmap
  - `links` are not currently supported on swarm stack deploys at present
