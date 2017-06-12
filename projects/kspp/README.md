# Kernel Self Protection Project (KSPP)

The [Kernel Self Protection Project](https://kernsec.org/wiki/index.php/Kernel_Self_Protection_Project) is a community
effort to harden the upstream Linux kernel by eliminating classes of vulnerabilities.

Many similar protections have existed in other projects, but have yet to have been upstreamed. Since Moby is a consumer
of the Linux kernel and aims to be the most secure distro it can be, it is in our maintainers' best interests to collaborate
on upstream Linux security measures.


## Roadmap

**Near-term:**
- We've aligned our `kernel_config` and `sysctl` settings with the
[KSPP recommendations](https://kernsec.org/wiki/index.php/Kernel_Self_Protection_Project#Recommended_settings) -
we should continue to track these
  - Note: we check for these settings in our CI tests (see `check_kernel_config.sh`)
- @tych0 is working on KSPP patches, which are submitted to the [kernel hardening mailing list](http://www.openwall.com/lists/kernel-hardening/)

**Long-term:**
- Increase involvement in the project
