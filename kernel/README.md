Temporary non modular kernel config for pinata alpha

TODO: build with Alpine and/or use Alpine kernels - needs some patches.

The build is mostly silent. To view the output use `docker log -f <containerid>`. The build creates multiple containers, so multiple
invocations may be necessary. To view the full build output one may also invoke `docker build .` and then copy the build artefacts from the image afterwards.

To build with various debug options enabled, build the kernel with
`make DEBUG=1`. The options enabled are listed in `kernel_config.debug`. This allocates a significant amount of memory on boot and you may need to adjust the kernel config on some systems. Specifically:
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
