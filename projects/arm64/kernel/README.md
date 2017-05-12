See [../docs/kernel-patches.md](../docs/kernels.md) for more
information on kernel builds.

To build with various debug options enabled, build the kernel with
`make DEBUG=1`. The options enabled are listed in `kernel_config.debug`.
This allocates a significant amount of memory on boot and you may need to
adjust the kernel config on some systems. Specifically:

```diff
--- a/alpine/kernel/kernel_config
+++ b/alpine/kernel/kernel_config
@@ -415,8 +415,8 @@ CONFIG_DMI=y
 # CONFIG_CALGARY_IOMMU is not set
 CONFIG_SWIOTLB=y
 CONFIG_IOMMU_HELPER=y
-CONFIG_MAXSMP=y
-CONFIG_NR_CPUS=8192
+CONFIG_MAXSMP=n
+CONFIG_NR_CPUS=8
 # CONFIG_SCHED_SMT is not set
 CONFIG_SCHED_MC=y
 # CONFIG_PREEMPT_NONE is not set
```
