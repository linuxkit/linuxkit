The incomplete list below is an assement of some CVEs, and Moby's resillience
to them.

Bugs mitigated:

* [CVE-2017-2636](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2017-2636)
  ([exploit post](https://a13xp0p0v.github.io/2017/03/24/CVE-2017-2636.html)):
  This CVE requires `CONFIG_N_HDLC={y|m}`, which Moby does not specify, and so
  is not vulnerable.
