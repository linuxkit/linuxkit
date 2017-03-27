# why do we care about dhcp clients?

DHCP allows a remote (but network-local) machine to specify configuration information to a host.  Many environments into which Moby might be deployed convey critical host configuration information (e.g. network settings) via DHCP, and use of this client will not be optional there.  An example of such an environment is Amazon EC2.

DHCP requires a client to have access to raw traffic from the network interface, as it necessarily precedes (and in fact causes) correct network configuration in the kernel.  DHCP clients also require configuration permissions for lots of things provided by the kernel, and so their position is doubly sensitive -- they are both privileged by necessity, as they must configure the system, and by circumstance, as they need additional privileges simply to do the network communication they require.

DHCP clients also carry an additional software burden.  As they can't directly use the usual sockets API for network traffic, most implement additional parsing and printing code for IP and UDP (which is usually done by the kernel for other network users).  These parsers are additional custom code on paths that are infrequently exercised, usually a recipe for bugs.

In summary, DHCP clients are at an unfortunate intersection of important, trusted, and complicated.

# how can we make them better?

We attempt to improve the situation for *any* DHCP client by implementing a privilege-separated model which runs the DHCP client within a system container.  This does not guard against misuse of the DHCP client's required capabilities, but mitigates attacks which manipulate the DHCP client into taking actions which aren't required in its normal mode of use.

We can also attempt to separate the two concerns of a DHCP client:  participating in network conversations that result in a valid lease, and using lease information to configure the system.  Separating these concerns and constraining the channel through which the information used to configure the system is expressed helps to mitigate attacks which trick the DHCP client into using its legitimate capabilities to do mischief.

# what more can we do?

Existing DHCP clients are generally written in memory-unsafe languages; their heavy use of parsers makes several common attack vectors promising.

MirageOS uses `charrua-client`, built on `charrua-core`, which depends lightly on `tcpip`; all OCaml libraries that attempt to replace memory-unsafe C with typed, memory-safe implementations.

As `charrua-client` is considerably less widely used than busybox's `udhcpcd` or ISC's `dhclient`, we attempt to demonstrate the trustworthiness of this replacement component with automated tools.  Within the scope of this analysis are client code adapted from `charrua-client`, the `Dhcp_wire` module of `charrua-core`, and the `Ethif_packet`, `Ipv4_packet`, and `Udp_packet` modules of `tcpip`.

## stuff we did

We used the well-regarded afl-fuzz to identify parsing bugs in the `tcpip` parser modules in question and fixed said bugs in [tcpip-parse-fixes](https://github.com/mirage/mirage-tcpip/pull/307).  We applied `afl-fuzz` to `charrua-core` and found no deficiencies in its DHCP parser.

## stuff we're doing

We will also use `afl-fuzz` directly to test the `charrua-client`-derived UNIX client's robustness when confronted with strange input.  This is sufficient only to find bugs which result in crashes or hangs; we are also using a novel tool [Crowbar](https://github.com/stedolan/crowbar) which combines AFL's instrumentation-guided fuzzing with [property-based testing](https://en.wikipedia.org/wik/QuickCheck) to discover inputs that cause the program to violate its stated properties.

We also intend to automatically test the client against a variety of commonly-used DHCP servers in various interesting configurations.  Servers we expect to test against include ISC's `dhcpd` and its successor `kea`, `dnsmasq`, and busybox `udhcpd`.
