# LinuxKit Security Events

The incomplete list below is an assessment of some CVEs, and LinuxKit's resilience
(or not) to them.

### Bugs mitigated:

* [CVE-2017-9075](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2017-9075):
  Requires CONFIG_IP_SCTP=y, which we do not set.
* [CVE-2017-9076](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2017-9076):
  Requires CONFIG_IP_DCCP=y, which we do not set. (However, we were vulnerable
  to the ipv6 pieces that this patch fixes.)
* [CVE-2017-1000363](http://www.openwall.com/lists/oss-security/2017/05/23/16):
  This CVE requires `CONFIG_PRINTER=y`, so we are not vulnerable.
* [CVE-2017-2636](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2017-2636)
  ([exploit post](https://a13xp0p0v.github.io/2017/03/24/CVE-2017-2636.html)):
  This CVE requires `CONFIG_N_HDLC={y|m}`, which LinuxKit does not specify, and so
  is not vulnerable.
* [CVE-2016-10229](http://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-10229)
  This CVE only applies to kernels `<= 4.5, <= 4.4.21`. By using recent kernels
  (specifically, kernels `=> 4.9, >= 4.4.21`, LinuxKit mitigates this bug.
* [CVE-2017-9605](http://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2017-9605):
  Requires CONFIG_DRM_VMWGFX=y, which we do not set.
* [CVE-2017-1000380](https://cve.mitre.org/cgi-bin/cvename.cgi?name=2017-1000380):
  Requires CONFIG_SOUND=y, which we do not set.
* [CVE-2017-7518](https://cve.mitre.org/cgi-bin/cvename.cgi?name=2017-7518):
  Requires the KVM backend (CONFIG_KVM=y), and we only have CONFIG_KVM_GUEST=y.
* [CVE-2017-10810](https://cve.mitre.org/cgi-bin/cvename.cgi?name=2017-10810)
  Requires CONFIG_DRM_VIRTIO_GPU, which we do not set.
* [CVE-2017-10911](https://cve.mitre.org/cgi-bin/cvename.cgi?name=2017-10911)
  aka XSA-216: we only have the XEN frontend, and do not set
  CONFIG_XEN_BLKDEV_BACKEND.

### Bugs fixed:

* [CVE-2017-8890](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2017-8890)
  All users can do `accept()`, mitigated for kernels `>= 4.9.31, >= 4.10.16, >= 4.11.2` now packaged by LinuxKit
* [CVE-2017-9077](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2017-9077)
  Same as CVE-2017-8890, but for ipv6.
* [CVE-2017-9074](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2017-9074):
  Users have access to ipv6 sockets, mitigated for kernels `>= 4.9.31, >= 4.10.16, >= 4.11.2` now packaged by LinuxKit
* [CVE-2017-9242](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2017-9242):
  Same as CVE-2017-9074.
* [CVE-2017-9076](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2017-9076):
  Users have access to ipv6 sockets (note that part of this is mitigated as
  well, so listed above: we do not set CONFIG_IP_DCCP), mitigated for kernels
  `>= 4.9.31, >= 4.10.16, >= 4.11.2` now packaged by LinuxKit
* [CVE-2017-1000364](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2017-1000364):
  [Qualys writeup](https://www.qualys.com/2017/06/19/stack-clash/stack-clash.txt).
  Fixed in kernels `>= 4.9.35, >= 4.11.8`, now packaged by LinuxKit.

### Bugs outstanding:
