# Weekly LinuxKit dev report for 2017-06-12 to 2017-06-18 (week 24)

This report covers weekly developments in the [linuxkit] [virtsock] and the [linuxkit-ci] repositories.
There is a [Moby development Summit](https://www.eventbrite.com/e/moby-summit-tickets-34483396768) in the
Docker office in San Francisco on June 19, with several of the LinuxKit developers present (see agenda at
[#2033]).  This week the
following major activity went into the tree:

**Added a static usermode helper:**: Linux 4.11 has a [safer mechanism](https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/commit/?id=64e90a8acb8590c2468c919f803652f081e3a4bf) for user mode helpers that forces all user-mode helper binaries to a single read-only path. Allowed binaries are whitelisted, and this reduces the attack surface in the kernel. ([#2037] [#1760] [@tych0] [@ijc] [@MagnusS] [@rn]).

**Moby command:** The tool now supports `~` in paths, allowing for example the user's ssh key to be automatically added in the ssh examples ([#2027] [@justincormack]). The `moby` command was also tidied up to use a unified coding style ([#2054] [@rn] [@riyazdf]).

**Dynamic VHD support:** There is now a mkimage package to create dynamic VHD images (static/fixed VHD images are already supported by LinuxKit). Dynamic VHD files are smaller in size, making them much easier to upload to the IBM cloud.  ([#1955] [@davefreitag] [@justincormack])

**Cold plug of devices:** While `mdev` handles hot-plug of devices added to the system after it was booted, it did not support cold-plug (i.e. loading modules for devices which are present on boot).  This is now supported via `rc.init` ([#2038] [@pwFoo][@rn] [@justincormack])

**Custom containerd client:** The latest containerd  has removed the `--runtime-config` option which we relied on. Since `ctr` is not (considered by containerd devs) to be a supported interface, LinuxKit now uses a custom client written against the containerd client library. ([#2041]  [@riyazdf] [@ijc] [@justincormack])

**setsid in init:** The containerisation of `getty` last week continues, with various improvements to support using `setsid` in the init phase as well as a service ([#2036] [#2044] [@deitch] [@riyazdf] [@ijc] [@rn] [@justincormack])

**Hyperkit multiple disk and vmnet:** Now that the Hyperkit Go API has multiple disk support, this is now available from LinuxKit as well. ([#2052] [@justincormack]). Vmnet support was also added to `linuxkit run hyperkit` to use the builtin OSX DHCP NAT ([#2060] [@justincormack]).

## Packaging

- **Kubernetes:** Updste to the latest init, combine the boot scripts into a single one, and give each instance a separate state directory. ([#2032] [@ijc] [@errordeveloper]  [@justincormack] [@riyazdf])
- **Docker for Mac:** A blueprint for the open source components of Docker for Mac is now in the tree. It includes support for VPNKit networking and port forwarding to the host. Docker can be controlled via a unix domain socket in the linuxkit state directory. ([#2039] [@MagnusS] [@ijc] [@rn] [@justincormack])
- **Docker CE**: Add a `vpnkit-expose-port` option ([#2048]  [@MagnusS] [@riyazdf] [@justincormack])
- Use `library` Hub org in examples to verify nginx, other official images ([#2059]  [@justincormack])

# Kernel and drivers

- Kernel has been updated to 4.11.5/4.9.32/4.4.72 + init update ([#2051] [@rn])
- USB drivers enabled on the 4.4.x, 4.9.x and 4.11.x kernels ([#2043] [@m4rcu5] [@nrocco] [@rn] [@justincormack])
- Remove kernel-compile and add perf package ([#2047] [@rn])

## Projects

- MirageSDK: replace custom transport protocol by Capnproto ([#2040] [@talex5] [@rn]), add an https example ([#1981] [@avsm] [@talex5] [@justincormack]) and work is continuing on making the DHCP client a dropin replacement for the current C version ([@samoht])

- A new Shiftfs project is available for mapping mountpoints across user namespaces ([#2035] [@tych0] [@estesp]  [@jejb] [@riyazdf])

## Docs

- Update security events with new kernels ([#2030] [@justincormack])
- Kernel config project docs ([#2042]  [@justincormack])
- Add Packet.net documentation ([#2057] [#2046] [@vielmetti] [@avsm])
- Update AUTHORS ([#2058] [@justincormack])

- Removed unused vendoring [#2050]  [@justincormack]
- Improve fetching of results [linuxkit-ci#8] [@talex5]

Other reports in this series can be browsed directly in the repository at [linuxkit:/reports](https://github.com/linuxkit/linuxkit/tree/master/reports/).

[@AkihiroSuda]: https://github.com/AkihiroSuda
[@Madko]: https://github.com/Madko
[@MagnusS]: https://github.com/MagnusS
[@alexellis]: https://github.com/alexellis
[@avsm]: https://github.com/avsm
[@davefreitag]: https://github.com/davefreitag
[@dcui]: https://github.com/dcui
[@deitch]: https://github.com/deitch
[@errordeveloper]: https://github.com/errordeveloper
[@estesp]: https://github.com/estesp
[@friism]: https://github.com/friism
[@furious-luke]: https://github.com/furious-luke
[@ijc]: https://github.com/ijc
[@jejb]: https://github.com/jejb
[@justincormack]: https://github.com/justincormack
[@m4rcu5]: https://github.com/m4rcu5
[@nrocco]: https://github.com/nrocco
[@pwFoo]: https://github.com/pwFoo
[@riyazdf]: https://github.com/riyazdf
[@rn]: https://github.com/rn
[@ryan-blunden]: https://github.com/ryan-blunden
[@s3ni0r]: https://github.com/s3ni0r
[@samoht]: https://github.com/samoht
[@talex5]: https://github.com/talex5
[@tha]: https://github.com/tha
[@thaJeztah]: https://github.com/thaJeztah
[@thebsdbox]: https://github.com/thebsdbox
[@tych0]: https://github.com/tych0
[@vielmetti]: https://github.com/vielmetti
[linuxkit]: https://github.com/linuxkit/linuxkit
[#1198]: https://github.com/linuxkit/linuxkit/issues/1198
[#1229]: https://github.com/linuxkit/linuxkit/issues/1229
[#1336]: https://github.com/linuxkit/linuxkit/issues/1336
[#1420]: https://github.com/linuxkit/linuxkit/issues/1420
[#1421]: https://github.com/linuxkit/linuxkit/issues/1421
[#1480]: https://github.com/linuxkit/linuxkit/issues/1480
[#1481]: https://github.com/linuxkit/linuxkit/issues/1481
[#1613]: https://github.com/linuxkit/linuxkit/issues/1613
[#1742]: https://github.com/linuxkit/linuxkit/issues/1742
[#1760]: https://github.com/linuxkit/linuxkit/issues/1760
[#1771]: https://github.com/linuxkit/linuxkit/issues/1771
[#1848]: https://github.com/linuxkit/linuxkit/issues/1848
[#1852]: https://github.com/linuxkit/linuxkit/issues/1852
[#1906]: https://github.com/linuxkit/linuxkit/pull/1906
[#1908]: https://github.com/linuxkit/linuxkit/pull/1908
[#1931]: https://github.com/linuxkit/linuxkit/issues/1931
[#1955]: https://github.com/linuxkit/linuxkit/pull/1955
[#1981]: https://github.com/linuxkit/linuxkit/pull/1981
[#1995]: https://github.com/linuxkit/linuxkit/issues/1995
[#2008]: https://github.com/linuxkit/linuxkit/pull/2008
[#2015]: https://github.com/linuxkit/linuxkit/issues/2015
[#2017]: https://github.com/linuxkit/linuxkit/pull/2017
[#2019]: https://github.com/linuxkit/linuxkit/issues/2019
[#2020]: https://github.com/linuxkit/linuxkit/issues/2020
[#2021]: https://github.com/linuxkit/linuxkit/pull/2021
[#2022]: https://github.com/linuxkit/linuxkit/pull/2022
[#2023]: https://github.com/linuxkit/linuxkit/pull/2023
[#2024]: https://github.com/linuxkit/linuxkit/pull/2024
[#2025]: https://github.com/linuxkit/linuxkit/pull/2025
[#2026]: https://github.com/linuxkit/linuxkit/pull/2026
[#2027]: https://github.com/linuxkit/linuxkit/pull/2027
[#2028]: https://github.com/linuxkit/linuxkit/pull/2028
[#2029]: https://github.com/linuxkit/linuxkit/pull/2029
[#2030]: https://github.com/linuxkit/linuxkit/pull/2030
[#2031]: https://github.com/linuxkit/linuxkit/issues/2031
[#2032]: https://github.com/linuxkit/linuxkit/pull/2032
[#2033]: https://github.com/linuxkit/linuxkit/issues/2033
[#2034]: https://github.com/linuxkit/linuxkit/issues/2034
[#2035]: https://github.com/linuxkit/linuxkit/pull/2035
[#2036]: https://github.com/linuxkit/linuxkit/pull/2036
[#2037]: https://github.com/linuxkit/linuxkit/pull/2037
[#2038]: https://github.com/linuxkit/linuxkit/pull/2038
[#2039]: https://github.com/linuxkit/linuxkit/pull/2039
[#2040]: https://github.com/linuxkit/linuxkit/pull/2040
[#2041]: https://github.com/linuxkit/linuxkit/pull/2041
[#2042]: https://github.com/linuxkit/linuxkit/pull/2042
[#2043]: https://github.com/linuxkit/linuxkit/pull/2043
[#2044]: https://github.com/linuxkit/linuxkit/pull/2044
[#2046]: https://github.com/linuxkit/linuxkit/issues/2046
[#2047]: https://github.com/linuxkit/linuxkit/pull/2047
[#2048]: https://github.com/linuxkit/linuxkit/pull/2048
[#2049]: https://github.com/linuxkit/linuxkit/issues/2049
[#2050]: https://github.com/linuxkit/linuxkit/pull/2050
[#2051]: https://github.com/linuxkit/linuxkit/pull/2051
[#2052]: https://github.com/linuxkit/linuxkit/pull/2052
[#2053]: https://github.com/linuxkit/linuxkit/issues/2053
[#2054]: https://github.com/linuxkit/linuxkit/pull/2054
[#2055]: https://github.com/linuxkit/linuxkit/issues/2055
[#2056]: https://github.com/linuxkit/linuxkit/issues/2056
[#2057]: https://github.com/linuxkit/linuxkit/pull/2057
[#2058]: https://github.com/linuxkit/linuxkit/pull/2058
[#2059]: https://github.com/linuxkit/linuxkit/pull/2059
[#2060]: https://github.com/linuxkit/linuxkit/pull/2060
[#2061]: https://github.com/linuxkit/linuxkit/issues/2061
[#2062]: https://github.com/linuxkit/linuxkit/pull/2062
[#2064]: https://github.com/linuxkit/linuxkit/issues/2064
[#2065]: https://github.com/linuxkit/linuxkit/issues/2065
[#2066]: https://github.com/linuxkit/linuxkit/pull/2066
[#2067]: https://github.com/linuxkit/linuxkit/issues/2067
[linuxkit-ci]: https://github.com/linuxkit/linuxkit-ci
[linuxkit-ci#10]: https://github.com/linuxkit/linuxkit-ci/pull/10
[linuxkit-ci#6]: https://github.com/linuxkit/linuxkit-ci/pull/6
[linuxkit-ci#8]: https://github.com/linuxkit/linuxkit-ci/pull/8
[linuxkit-ci#9]: https://github.com/linuxkit/linuxkit-ci/pull/9
[virtsock]: https://github.com/linuxkit/virtsock
