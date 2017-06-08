#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

int main(int argc, char *argv[])
{
	int i;

	/* TODO: this doesn't go anywhere useful right now. It would be nice to
	 * switch this to syslog() (or some other mechanism) so that we can
	 * actually read the contents.
	 */
	fprintf(stderr, "usermodehelper: ");
	for (i = 0; i < argc; i++) {
		fprintf(stderr, "%s ", argv[i]);
	}
	fprintf(stderr, "\n");

	if (!strcmp(argv[0], "/sbin/mdev")) {
		/* busybox uses /sbin/mdev for early uevent bootstrapping */
		execv(argv[0], argv);
	} else if (!strcmp(argv[0], "/sbin/modprobe")) {
		/* allow modprobe */
		execv(argv[0], argv);
	} else if (!strcmp(argv[0], "/sbin/poweroff") ||
			!strcmp(argv[0], "/sbin/reboot")) {
		/* poweroff and reboot are allowed */
		execv(argv[0], argv);
	} else {
		/* This means either we got an unexpected call from the kernel
		 * or someone is doing something nefarious. Some other possible
		 * expected callers are:
		 *  - for core dumps. we don't have a "core" binary, and don't
		 *    set this by default to anything. when we do, we need to
		 *    whitelist it here
		 *  - /linuxrc: we're not doing legacy root setup, so we don't
		 *     need this
		 *  - a few drivers and filesystems (drbd, nfs, nfsd, ocfs2)
		 *  - cgroup notify_on_release handlers, which we do not set
		 *    (but e.g. systemd needs, if anyone ever tries to boot
		 *    that on linuxkit)
		 *  - /sbin/request-key, which we don't provide
		 *  - on x86, machine check
		 *
		 * Today we only call mdev and modprobe, but as we add more
		 * features to linuxkit this whitelist may need changing (or a
		 * policy, like always allow stuff in /sbin).
		 */
		exit(2);
	}

	perror("exec failed");
	return 1;
}
