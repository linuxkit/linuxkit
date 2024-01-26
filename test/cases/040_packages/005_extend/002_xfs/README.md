# xfs tests

XFS packages - xfsprogs - is generally forward compatible but not backwards compatible.
This means that a more recent version of xfsprogs will not work with an older
kernel.

To avoid this issue, do not update these tests unless you are also updating the
kernel.

This can be made simpler by having kernel-specific versions.
