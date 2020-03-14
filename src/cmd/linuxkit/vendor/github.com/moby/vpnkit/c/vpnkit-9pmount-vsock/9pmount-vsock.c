#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <getopt.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <errno.h>
#include <unistd.h>
#include <fcntl.h>
#include <err.h>
#include <sys/wait.h>
#include <sys/socket.h>
#include <linux/vm_sockets.h>

#include "hvsock.h"
#include "log.h"

#define NONE 0
#define LISTEN 1
#define CONNECT 2

int mode = NONE;

char *mount = "/bin/mount";

static int handle(int fd, char *tag, char *path)
{
	char *options = NULL;
	int status;
	pid_t pid;
	int res;

	res = asprintf(&options,
		       "trans=fd,dfltuid=1001,dfltgid=50,version=9p2000,msize=4096,rfdno=%d,wfdno=%d",
		       fd, fd);
	if (res < 0)
		fatal("asprintf()");

	char *argv[] = {
		mount,
		"-t", "9p", "-o", options,
		tag, path,
		NULL
	};

	pid = fork();
	if (pid == 0) {
		execv(mount, argv);
		fatal("execv()");
	}

	res = waitpid(pid, &status, 0);
	if (res == -1) {
		ERROR("waitpid failed: %d. %s", errno, strerror(errno));
		exit(1);
	}
	return WEXITSTATUS(status);
}

static int create_listening_hvsocket(GUID serviceid)
{
	SOCKADDR_HV sa;
	int lsock;
	int res;

	lsock = socket(AF_HYPERV, SOCK_STREAM, HV_PROTOCOL_RAW);
	if (lsock == -1)
		return -1;

	bzero(&sa, sizeof(sa));
	sa.Family = AF_HYPERV;
	sa.Reserved = 0;
	sa.VmId = HV_GUID_WILDCARD;
	sa.ServiceId = serviceid;

	res = bind(lsock, (const struct sockaddr *)&sa, sizeof(sa));
	if (res == -1)
		return -1; /* ignore the fd leak */

	res = listen(lsock, 1);
	if (res == -1)
		return -1; /* ignore the fd leak */

	return lsock;
}

static int create_listening_vsocket(long port)
{
	struct sockaddr_vm sa;
	int lsock;
	int res;

	lsock = socket(AF_VSOCK, SOCK_STREAM, 0);
	if (lsock == -1)
		return -1;

	bzero(&sa, sizeof(sa));
	sa.svm_family = AF_VSOCK;
	sa.svm_reserved1 = 0;
	sa.svm_port = port;
	sa.svm_cid = VMADDR_CID_ANY;

	res = bind(lsock, (const struct sockaddr *)&sa, sizeof(sa));
	if (res == -1)
		return -1; /* ignore the fd leak */

	res = listen(lsock, 1);
	if (res == -1)
		return -1; /* ignore the fd leak */

	return lsock;
}


static int connect_hvsocket(GUID serviceid)
{
	SOCKADDR_HV sa;
	int sock;
	int res;

	sock = socket(AF_HYPERV, SOCK_STREAM, HV_PROTOCOL_RAW);
	if (sock == -1)
		return -1;

	bzero(&sa, sizeof(sa));
	sa.Family = AF_HYPERV;
	sa.Reserved = 0;
	sa.VmId = HV_GUID_PARENT;
	sa.ServiceId = serviceid;

	res = connect(sock, (const struct sockaddr *)&sa, sizeof(sa));
	if (res == -1)
		return -1; /* ignore the fd leak */

	return sock;
}

static int connect_vsocket(long port)
{
	struct sockaddr_vm sa;
	int sock;
	int res;

	sock = socket(AF_VSOCK, SOCK_STREAM, 0);
	if (sock == -1)
		return -1;

	bzero(&sa, sizeof(sa));
	sa.svm_family = AF_VSOCK;
	sa.svm_reserved1 = 0;
	sa.svm_port = port;
	sa.svm_cid = VMADDR_CID_HOST;

	res = connect(sock, (const struct sockaddr *)&sa, sizeof(sa));
	if (res == -1)
		return -1; /* ignore the fd leak */

	return sock;
}

static int accept_hvsocket(int lsock)
{
	SOCKADDR_HV sac;
	socklen_t socklen = sizeof(sac);
	int csock;

	csock = accept(lsock, (struct sockaddr *)&sac, &socklen);
	if (csock == -1)
		fatal("accept()");

	INFO("Connect from: " GUID_FMT ":" GUID_FMT "\n",
	       GUID_ARGS(sac.VmId), GUID_ARGS(sac.ServiceId));

	return csock;
}

static int accept_vsocket(int lsock)
{
	struct sockaddr_vm sac;
	socklen_t socklen = sizeof(sac);
	int csock;

	csock = accept(lsock, (struct sockaddr *)&sac, &socklen);
	if (csock == -1)
		fatal("accept()");

	INFO("Connect from: port=%x cid=%d", sac.svm_port, sac.svm_cid);

	return csock;
}

void usage(char *name)
{
	printf("%s: mount a 9P filesystem from an hvsock connection\n", name);
	printf("usage:\n");
	printf("\t[--vsock port] <listen | connect> <tag> <path>\n");
	printf("where\n");
	printf("\t--vsock <port>: use the AF_VSOCK <port>\n");
	printf("\t--listen: listen forever for incoming AF_VSOCK connections\n");
	printf("\t--connect: connect to the parent partition\n");
}

int main(int argc, char **argv)
{
	int res = 0;
	GUID sid;
	int c;
	unsigned int port = 0;
	char serviceid[37]; /* 36 for a GUID and 1 for a NULL */
	char *tag = NULL;
	char *path = NULL;

	opterr = 0;
	while (1) {
		static struct option long_options[] = {
			/* These options set a flag. */
			{"vsock", required_argument, NULL, 'v'},
			{"verbose", no_argument, NULL, 'w'},
			{0, 0, 0, 0}
		};
		int option_index = 0;

		c = getopt_long(argc, argv, "v:", long_options, &option_index);
		if (c == -1)
			break;

		switch (c) {
		case 'v':
			port = (unsigned int) strtol(optarg, NULL, 0);
			break;
		case 'w':
			verbose++;
			break;
		case 0:
			break;
		default:
			usage(argv[0]);
			exit(1);
		}
	}

	if (optind < argc) {
		if (strcmp(argv[optind], "listen") == 0)
			mode = LISTEN;
		else if (strcmp(argv[optind], "connect") == 0)
			mode = CONNECT;
		optind++;
	}
	if (mode == NONE) {
		fprintf(stderr, "Please supply either listen or connect\n");
		usage(argv[0]);
		exit(1);
	}

	if (optind < argc)
		tag = argv[optind++];

	if (optind < argc)
		path = argv[optind++];

	if (!tag) {
		fprintf(stderr, "Please supply a tag name\n");
		usage(argv[0]);
		exit(1);
	}

	if (!path) {
		fprintf(stderr, "Please supply a path\n");
		usage(argv[0]);
		exit(1);
	}

	snprintf(serviceid, sizeof(serviceid), "%08x-FACB-11E6-BD58-64006A7986D3", port);
	res = parseguid(serviceid, &sid);
	if (res) {
		fprintf(stderr,
			"Failed to parse serviceid as GUID: %s\n", serviceid);
		usage(argv[0]);
		exit(1);
	}

	for (;;) {
		int lsocket;
		int sock;
		int r;

		if (mode == LISTEN) {
			INFO("starting in listening mode with port=%x, tag=%s, path=%s", port, tag, path);
			lsocket = create_listening_vsocket(port);
			if (lsocket != -1) {
				sock = accept_vsocket(lsocket);
				close(lsocket);
			} else {
				INFO("failed to create AF_VSOCK, trying with AF_HVSOCK serviceid=%s", serviceid);
				lsocket = create_listening_hvsocket(sid);
				if (lsocket == -1)
					fatal("create_listening_vsocket");
				sock = accept_hvsocket(lsocket);
				close(lsocket);
			}
		} else {
			INFO("starting in connect mode with port=%x, tag=%s, path=%s", port, tag, path);
			sock = connect_vsocket(port);
			if (sock == -1) {
				INFO("failed to connect AF_VSOCK, trying with AF_HVSOCK serviceid=%s", serviceid);
				sock = connect_hvsocket(sid);
			}
		}

		r = handle(sock, tag, path);
		close(sock);

		if (r == 0) {
			INFO("mount successful for (serviceid=%s) port=%x tag=%s path=%s", serviceid, port, tag, path);
			exit(0);
		}

		/*
		 * This can happen if the client times out the connection
		 * after we accept it
		 */
		ERROR("mount failed with %d for (serviceid=%s) port=%x tag=%s path=%s", r, serviceid, port, tag, path);
		sleep(1); /* retry */
	}
}
