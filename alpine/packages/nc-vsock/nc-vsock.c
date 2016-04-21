#include <errno.h>
#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <stdbool.h>
#include <unistd.h>
#include <fcntl.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/select.h>
#include <netdb.h>
#include <uuid/uuid.h>
#include "include/uapi/linux/vm_sockets.h"

#define MODE_READ 1 /* From the vsock */
#define MODE_WRITE 2 /* To the vsock */
#define MODE_RDWR (MODE_READ|MODE_WRITE)

/*
 * Hyper-V Sockets headerfile pull in too much other stuff. Replicate
 * the bits we need here.
 */
#ifndef AF_HYPERV
#define AF_HYPERV 42
#endif

struct sockaddr_hv {
	unsigned short  shv_family;          /* Address family          */
	unsigned short  reserved;            /* Must be Zero            */
	uuid_t          shv_vm_id;           /* Not used. Must be Zero. */
	uuid_t          shv_service_id;      /* Service ID              */
};
UUID_DEFINE(SHV_VMID_GUEST,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0);
#define SHV_PROTO_RAW 1

/*
 * MSFT's GUIDs are a bonkers mix of native and big endian byte
 * order. The uuid library uses RFC 4122, which is always big endian.
 * The Linux kernel uuid.h actually looks more like it should be
 * called guid.h. We use the uuid library for ease of parsing/printing
 * and then this function to convert between UUID and GUID.
 * https://en.wikipedia.org/wiki/Globally_unique_identifier
 */
static void uuid2guid(uuid_t u)
{
#if __BYTE_ORDER__ == __ORDER_LITTLE_ENDIAN__
	char t;
	t = u[0]; u[0] = u[3]; u[3] = t;
	t = u[1]; u[1] = u[2]; u[2] = t;

	t = u[4]; u[4] = u[5]; u[5] = t;

	t = u[6]; u[6] = u[7]; u[7] = t;
#endif
}

static int parse_cid(const char *cid_str)
{
	char *end = NULL;
	long cid = strtol(cid_str, &end, 10);

	if (cid_str != end && *end == '\0') {
		return cid;
	} else {
		fprintf(stderr, "invalid cid: %s\n", cid_str);
		return -1;
	}
}

static int parse_port(const char *port_str)
{
	char *end = NULL;
	long port = strtol(port_str, &end, 10);

	if (port_str != end && *end == '\0') {
		return port;
	} else {
		fprintf(stderr, "invalid port number: %s\n", port_str);
		return -1;
	}
}

static int vsock_listen(const char *port_str)
{
	int listen_fd;
	int client_fd;
	struct sockaddr_vm sa_listen = {
		.svm_family = AF_VSOCK,
		.svm_cid = VMADDR_CID_ANY,
	};
	struct sockaddr_vm sa_client;
	socklen_t socklen_client = sizeof(sa_client);
	int port;
	int ret;

	port = parse_port(port_str);
	if (port < 0)
		return -1;

	sa_listen.svm_port = port;

	listen_fd = socket(AF_VSOCK, SOCK_STREAM, 0);
	if (listen_fd < 0) {
		perror("socket");
		return -1;
	}

	ret = bind(listen_fd, (struct sockaddr*)&sa_listen, sizeof(sa_listen));
	if (ret != 0) {
		perror("bind");
		close(listen_fd);
		return -1;
	}

	ret = listen(listen_fd, 1);
	if (ret != 0) {
		perror("listen");
		close(listen_fd);
		return -1;
	}

	client_fd = accept(listen_fd,
		(struct sockaddr*)&sa_client, &socklen_client);
	if (client_fd < 0) {
		perror("accept");
		close(listen_fd);
		return -1;
	}

	fprintf(stderr, "Connection from cid %u port %u...\n",
		sa_client.svm_cid, sa_client.svm_port);

	close(listen_fd);
	return client_fd;
}

static int hvsock_listen(const char *port_str)
{
	int listen_fd;
	int client_fd;
	struct sockaddr_hv sa_listen = {
		.shv_family = AF_HYPERV,
		.reserved = 0,
	};
	struct sockaddr_hv sa_client;
	socklen_t socklen_client = sizeof(sa_client);
	char vm_str[128], svc_str[128];
	int ret;

	uuid_copy(sa_listen.shv_vm_id, SHV_VMID_GUEST);

	ret = uuid_parse(port_str, sa_listen.shv_service_id);
	if (ret != 0)
		return -1;

	uuid2guid(sa_listen.shv_service_id);

	listen_fd = socket(AF_HYPERV, SOCK_STREAM, SHV_PROTO_RAW);
	if (listen_fd < 0) {
		perror("socket");
		return -1;
	}

	ret = bind(listen_fd, (struct sockaddr*)&sa_listen, sizeof(sa_listen));
	if (ret != 0) {
		perror("bind");
		close(listen_fd);
		return -1;
	}

	ret = listen(listen_fd, 1);
	if (ret != 0) {
		perror("listen");
		close(listen_fd);
		return -1;
	}

	client_fd = accept(listen_fd,
			   (struct sockaddr*)&sa_client, &socklen_client);
	if (client_fd < 0) {
		perror("accept");
		close(listen_fd);
		return -1;
	}

	uuid_unparse(sa_client.shv_vm_id, vm_str);
	uuid_unparse(sa_client.shv_service_id, svc_str);
	fprintf(stderr, "Connection from %s port %s...\n", vm_str, svc_str);

	close(listen_fd);
	return client_fd;
}

static int tcp_connect(const char *node, const char *service)
{
	int fd;
	int ret;
	const struct addrinfo hints = {
		.ai_family = AF_INET,
		.ai_socktype = SOCK_STREAM,
	};
	struct addrinfo *res = NULL;
	struct addrinfo *addrinfo;

	ret = getaddrinfo(node, service, &hints, &res);
	if (ret != 0) {
		fprintf(stderr, "getaddrinfo failed: %s\n", gai_strerror(ret));
		return -1;
	}

	for (addrinfo = res; addrinfo; addrinfo = addrinfo->ai_next) {
		fd = socket(addrinfo->ai_family,
			    addrinfo->ai_socktype, addrinfo->ai_protocol);
		if (fd < 0) {
			perror("socket");
			continue;
		}

		ret = connect(fd, addrinfo->ai_addr, addrinfo->ai_addrlen);
		if (ret != 0) {
			perror("connect");
			close(fd);
			continue;
		}

		break;
	}

	freeaddrinfo(res);
	return fd;
}

static int vsock_connect(const char *cid_str, const char *port_str)
{
	int fd;
	int cid;
	int port;
	struct sockaddr_vm sa = {
		.svm_family = AF_VSOCK,
	};
	int ret;

	cid = parse_cid(cid_str);
	if (cid < 0)
		return -1;

	sa.svm_cid = cid;

	port = parse_port(port_str);
	if (port < 0)
		return -1;

	sa.svm_port = port;

	fd = socket(AF_VSOCK, SOCK_STREAM, 0);
	if (fd < 0) {
		perror("socket");
		return -1;
	}

	ret = connect(fd, (struct sockaddr*)&sa, sizeof(sa));
	if (ret != 0) {
		perror("connect");
		close(fd);
		return -1;
	}

	return fd;
}

static int hvsock_connect(const char *vm_str, const char *svc_str)
{
	int fd;
	int ret;
	struct sockaddr_hv sa = {
		.shv_family = AF_HYPERV,
		.reserved = 0,
	};

	ret = uuid_parse(vm_str, sa.shv_vm_id);
	if (ret != 0) {
		fprintf(stderr, "VM GUID parse error: %s\n", vm_str);
		return -1;
	}
	uuid2guid(sa.shv_vm_id);

	ret = uuid_parse(svc_str, sa.shv_service_id);
	if (ret != 0) {
		fprintf(stderr, "Service GUID parse error: %s\n", svc_str);
		return -1;
	}
	uuid2guid(sa.shv_service_id);

	fd = socket(AF_HYPERV, SOCK_STREAM, SHV_PROTO_RAW);
	if (fd < 0) {
		perror("socket");
		return -1;
	}

	ret = connect(fd, (struct sockaddr*)&sa, sizeof(sa));
	if (ret != 0) {
		perror("connect");
		close(fd);
		return -1;
	}

	return fd;
}

static int get_fds(int argc, char **argv, int fds[2])
{
	fds[0] = STDIN_FILENO;
	fds[1] = -1;

	if (argc >= 3 && strcmp(argv[1], "-l") == 0) {
		if (strstr(argv[2], "-"))
			fds[1] = hvsock_listen(argv[2]);
		else
			fds[1] = vsock_listen(argv[2]);
		if (fds[1] < 0)
			return -1;

		if (argc == 6 && strcmp(argv[3], "-t") == 0) {
			fds[0] = tcp_connect(argv[4], argv[5]);
			if (fds[0] < 0) {
				return -1;
			}
		}
		return 0;
	} else if (argc == 3) {
		if (strstr(argv[1], "-") || strstr(argv[2], "-"))
			fds[1] = hvsock_connect(argv[1], argv[2]);
		else
			fds[1] = vsock_connect(argv[1], argv[2]);
		if (fds[1] < 0)
			return -1;
		return 0;
	} else {
		fprintf(stderr, "usage: %s [-r|-w] [-l <port> [-t <dst> <dstport>] | <cid> <port>]\n", argv[0]);
		return -1;
	}
}

static void set_nonblock(int fd, bool enable)
{
	int ret;
	int flags;

	ret = fcntl(fd, F_GETFL);
	if (ret < 0) {
		perror("fcntl");
		return;
	}

	flags = ret & ~O_NONBLOCK;
	if (enable)
		flags |= O_NONBLOCK;

	fcntl(fd, F_SETFL, flags);
}

static int xfer_data(int in_fd, int out_fd)
{
	char buf[256*1024];
	char *send_ptr = buf;
	ssize_t nbytes;
	ssize_t remaining;
	int ret;

	if (out_fd == STDIN_FILENO) out_fd = STDOUT_FILENO;

	nbytes = read(in_fd, buf, sizeof(buf));
	if (nbytes < 0)
		return -1;

	if (nbytes == 0) {
		if (out_fd == STDOUT_FILENO)
			return 0;
		ret = shutdown(out_fd, SHUT_WR);
		if (ret == 0)
			return 0;
		perror("shutdown");
		return -1;
	}

	remaining = nbytes;
	while (remaining > 0) {
		nbytes = write(out_fd, send_ptr, remaining);
		if (nbytes < 0 && errno == EAGAIN)
			nbytes = 0;
		else if (nbytes <= 0)
			return -1;

		if (remaining > nbytes) {
			/* Wait for fd to become writeable again */
			for (;;) {
				fd_set wfds;
				FD_ZERO(&wfds);
				FD_SET(out_fd, &wfds);
				ret = select(out_fd + 1, NULL,
					     &wfds, NULL, NULL);
				if (ret < 0) {
					if (errno == EINTR) {
						continue;
					} else {
						perror("select");
						return -1;
					}
				}

				if (FD_ISSET(out_fd, &wfds))
					break;
			}
		}

		send_ptr += nbytes;
		remaining -= nbytes;
	}
	return 1;
}

static void main_loop(int fds[2], int mode)
{
	fd_set rfds;
	int nfds = fds[fds[0] > fds[1] ? 0 : 1] + 1;
	/* Which fd's are readable */
	bool rfd0 = !!(mode&MODE_WRITE), rfd1 = !!(mode&MODE_READ);
	int ret;

	set_nonblock(fds[0], true);
	set_nonblock(fds[1], true);

	for (;;) {
		if (!rfd0 && !rfd1)
			return;

		FD_ZERO(&rfds);
		if (rfd0)
			FD_SET(fds[0], &rfds);
		if (rfd1)
			FD_SET(fds[1], &rfds);

		ret = select(nfds, &rfds, NULL, NULL, NULL);
		if (ret < 0) {
			if (errno == EINTR) {
				continue;
			} else {
				perror("select");
				return;
			}
		}

		if (rfd0 && FD_ISSET(fds[0], &rfds)) {
			switch (xfer_data(fds[0], fds[1])) {
			case -1: return;
			case 0: rfd0 = false; break;
			case 1: break;
			}
		}

		if (rfd1 && FD_ISSET(fds[1], &rfds)) {
			switch (xfer_data(fds[1], fds[0])) {
			case -1: return;
			case 0: rfd1 = false; break;
			case 1: break;
			}
		}
	}
}

int main(int argc, char **argv)
{
	int mode = MODE_RDWR;
	int fds[2];

	if (argc >= 2) {
		if (!strcmp(argv[1], "-r")) {
			mode = MODE_READ;
			argv++; argc--;
		} else if (!strcmp(argv[1], "-w")) {
			mode = MODE_WRITE;
			argv++; argc--;
		}
	}

	if (get_fds(argc, argv, fds) < 0)
		return EXIT_FAILURE;

	main_loop(fds, mode);
	return EXIT_SUCCESS;
}
