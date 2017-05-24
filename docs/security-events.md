# LinuxKit Security Events

The incomplete list below is an assessment of some CVEs, and LinuxKit's resilience
(or not) to them.

### Bugs mitigated:

* [CVE-2017-1000363](http://www.openwall.com/lists/oss-security/2017/05/23/16):
  This CVE requires `CONFIG_PRINTER=y`, so we are not vulnerable.
* [CVE-2017-2636](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2017-2636)
  ([exploit post](https://a13xp0p0v.github.io/2017/03/24/CVE-2017-2636.html)):
  This CVE requires `CONFIG_N_HDLC={y|m}`, which LinuxKit does not specify, and so
  is not vulnerable.
* [CVE-2016-10229](http://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-10229)
  This CVE only applies to kernels `<= 4.5, <= 4.4.21`. By using recent kernels
  (specifically, kernels `=> 4.9, >= 4.4.21`, LinuxKit mitigates this bug.

### Bugs not mitigated:
