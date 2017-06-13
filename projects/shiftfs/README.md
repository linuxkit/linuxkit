## shiftfs

Shiftfs is a virtual filesystem for mapping mountpoints across user namespaces.
The idea is that it would be useful for dockerds spawning containers: they can
keep filesystems on the host disk in terms of real root, but mount the
container roots via shiftfs, allowing containers to share a particular
filesystem with different uid maps, while not having to uidshift every file on
disk (and thus destroying some of the sharing properties).

The version included here is the v2 version of shiftfs, using the superblock's
user namespace instead of mountopts to figure out mappings. Thus, an extra step
of "marking" mounts is needed. For example:

    # mkdir source
    # touch source/foo  # a root owned file
    # mount -t shiftfs -o mark source source
    # chmod 777 source

Now, let's make a user namespace:

    # setuid 1000 unshare -rm
    # cat /proc/self/uidmap
             0       1000          1
    # mkdir dest
    # mount -t shiftfs source dest
    # stat dest/foo | grep Uid
    Access: (0644/-rw-r--r--)  Uid: (    0/    root)   Gid: (    0/    root)

And thanks to the magic of shiftfs, the file is root owned in the user
namespace.
