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
#include "include/uapi/linux/vm_sockets.h"

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
	int port = parse_port(port_str);
	if (port < 0) {
		return -1;
	}

	sa_listen.svm_port = port;

	listen_fd = socket(AF_VSOCK, SOCK_STREAM, 0);
	if (listen_fd < 0) {
		perror("socket");
		return -1;
	}

	if (bind(listen_fd, (struct sockaddr*)&sa_listen, sizeof(sa_listen)) != 0) {
		perror("bind");
		close(listen_fd);
		return -1;
	}

	if (listen(listen_fd, 1) != 0) {
		perror("listen");
		close(listen_fd);
		return -1;
	}

	client_fd = accept(listen_fd, (struct sockaddr*)&sa_client, &socklen_client);
	if (client_fd < 0) {
		perror("accept");
		close(listen_fd);
		return -1;
	}

	fprintf(stderr, "Connection from cid %u port %u...\n", sa_client.svm_cid, sa_client.svm_port);

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
		fd = socket(addrinfo->ai_family, addrinfo->ai_socktype, addrinfo->ai_protocol);
		if (fd < 0) {
			perror("socket");
			continue;
		}

		if (connect(fd, addrinfo->ai_addr, addrinfo->ai_addrlen) != 0) {
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

	cid = parse_cid(cid_str);
	if (cid < 0) {
		return -1;
	}
	sa.svm_cid = cid;

	port = parse_port(port_str);
	if (port < 0) {
		return -1;
	}
	sa.svm_port = port;

	fd = socket(AF_VSOCK, SOCK_STREAM, 0);
	if (fd < 0) {
		perror("socket");
		return -1;
	}

	if (connect(fd, (struct sockaddr*)&sa, sizeof(sa)) != 0) {
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
		fds[1] = vsock_listen(argv[2]);
		if (fds[1] < 0) {
			return -1;
		}

		if (argc == 6 && strcmp(argv[3], "-t") == 0) {
			fds[0] = tcp_connect(argv[4], argv[5]);
			if (fds[0] < 0) {
				return -1;
			}
		}
		return 0;
	} else if (argc == 3) {
		fds[1] = vsock_connect(argv[1], argv[2]);
		if (fds[1] < 0) {
			return -1;
		}
		return 0;
	} else {
		fprintf(stderr, "usage: %s [-l <port> [-t <dst> <dstport>] | <cid> <port>]\n", argv[0]);
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
	if (enable) {
		flags |= O_NONBLOCK;
	}

	fcntl(fd, F_SETFL, flags);
}

static int xfer_data(int in_fd, int out_fd)
{
	char buf[4096];
	char *send_ptr = buf;
	ssize_t nbytes;
	ssize_t remaining;

	nbytes = read(in_fd, buf, sizeof(buf));
	if (nbytes <= 0) {
		return -1;
	}

	remaining = nbytes;
	while (remaining > 0) {
		nbytes = write(out_fd, send_ptr, remaining);
		if (nbytes < 0 && errno == EAGAIN) {
			nbytes = 0;
		} else if (nbytes <= 0) {
			return -1;
		}

		if (remaining > nbytes) {
			/* Wait for fd to become writeable again */
			for (;;) {
				fd_set wfds;
				FD_ZERO(&wfds);
				FD_SET(out_fd, &wfds);
				if (select(out_fd + 1, NULL, &wfds, NULL, NULL) < 0) {
					if (errno == EINTR) {
						continue;
					} else {
						perror("select");
						return -1;
					}
				}

				if (FD_ISSET(out_fd, &wfds)) {
					break;
				}
			}
		}

		send_ptr += nbytes;
		remaining -= nbytes;
	}
	return 0;
}

static void main_loop(int fds[2])
{
	fd_set rfds;
	int nfds = fds[fds[0] > fds[1] ? 0 : 1] + 1;

	set_nonblock(fds[0], true);
	set_nonblock(fds[1], true);

	for (;;) {
		FD_ZERO(&rfds);
		FD_SET(fds[0], &rfds);
		FD_SET(fds[1], &rfds);

		if (select(nfds, &rfds, NULL, NULL, NULL) < 0) {
			if (errno == EINTR) {
				continue;
			} else {
				perror("select");
				return;
			}
		}

		if (FD_ISSET(fds[0], &rfds)) {
			if (xfer_data(fds[0], fds[1]) < 0) {
				return;
			}
		}

		if (FD_ISSET(fds[1], &rfds)) {
			if (xfer_data(fds[1], fds[0]) < 0) {
				return;
			}
		}
	}
}

int main(int argc, char **argv)
{
	int fds[2];

	if (get_fds(argc, argv, fds) < 0) {
		return EXIT_FAILURE;
	}

	main_loop(fds);
	return EXIT_SUCCESS;
}
