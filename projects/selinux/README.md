# selinux

The ultimate goal here is to use SELinux as our default LSM in Moby. To this
end, here are the compiler flags and userspace packages necessary to do the
basics.

# TODO

All the necessary binaries exist, so the next steps are:

* label the filesystem with a default label
* have a policy that contains containerd
* label each container's files seprately, and contain them each with a policy
* policies for other system daemons
