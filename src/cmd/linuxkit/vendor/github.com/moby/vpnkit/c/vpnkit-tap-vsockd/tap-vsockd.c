#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <getopt.h>
#include <syslog.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <errno.h>
#include <unistd.h>
#include <fcntl.h>
#include <err.h>
#include <sys/uio.h>
#include <stdint.h>
#include <pthread.h>
#include <arpa/inet.h>
#include <sys/ioctl.h>
#include <net/if.h>
#include <linux/if_tun.h>
#include <net/if_arp.h>
#include <assert.h>

#include <sys/types.h>
#include <sys/socket.h>
#include <sys/wait.h>
#include <ifaddrs.h>

#include "hvsock.h"
#include "protocol.h"
#include "ring.h"

int daemon_flag;
int nofork_flag;
int listen_flag;
int connect_flag;

char *default_sid = "30D48B34-7D27-4B0B-AAAF-BBBED334DD59";

/* Support big frames if the server requests it */
const int max_packet_size = 16384;

static int verbose;
#define INFO(...)                                                       \
    do {                                                                \
        if (verbose) {                                                  \
            printf(__VA_ARGS__);                                        \
            fflush(stdout);                                             \
        }                                                               \
    } while (0)
#define DBG(...)                                                        \
    do {                                                                \
        if (verbose > 1) {                                              \
            printf(__VA_ARGS__);                                        \
            fflush(stdout);                                             \
        }                                                               \
    } while (0)
#define TRC(...)                                                        \
    do {                                                                \
        if (verbose > 2) {                                              \
            printf(__VA_ARGS__);                                        \
            fflush(stdout);                                             \
        }                                                               \
    } while (0)

void fatal(const char *msg)
{
	syslog(LOG_CRIT, "%s Error: %d. %s", msg, errno, strerror(errno));
	exit(1);
}

int alloc_tap(const char *dev)
{
	const char *clonedev = "/dev/net/tun";
	struct ifreq ifr;
	int persist = 1;
	int fd;

	fd = open(clonedev, O_RDWR);
	if (fd == -1)
		fatal("Failed to open /dev/net/tun");

	memset(&ifr, 0, sizeof(ifr));
	ifr.ifr_flags = IFF_TAP | IFF_NO_PI;
	strncpy(ifr.ifr_name, dev, IFNAMSIZ);
	if (ioctl(fd, TUNSETIFF, (void *)&ifr) < 0)
		fatal("TUNSETIFF failed");

	if (ioctl(fd, TUNSETPERSIST, persist) < 0)
		fatal("TUNSETPERSIST failed");

	syslog(LOG_INFO, "successfully created TAP device %s", dev);
	return fd;
}

void set_macaddr(const char *dev, uint8_t *mac)
{
	struct ifreq ifq;
	int fd;

	fd = socket(PF_INET, SOCK_DGRAM, 0);
	if (fd == -1)
		fatal("Could not get socket to set MAC address");
	strcpy(ifq.ifr_name, dev);
	memcpy(&ifq.ifr_hwaddr.sa_data[0], mac, 6);
	ifq.ifr_hwaddr.sa_family = ARPHRD_ETHER;

	if (ioctl(fd, SIOCSIFHWADDR, &ifq) == -1)
		fatal("SIOCSIFHWADDR failed");

	close(fd);
}

void set_mtu(const char *dev, int mtu)
{
	struct ifreq ifq;
	int fd;

	fd = socket(PF_INET, SOCK_DGRAM, 0);
	if (fd == -1)
		fatal("Could not get socket to set MTU");
	strcpy(ifq.ifr_name, dev);
	ifq.ifr_mtu = mtu;

	if (ioctl(fd, SIOCSIFMTU, &ifq) == -1)
		fatal("SIOCSIFMTU failed");

	close(fd);
}

/* Negotiate a vmnet connection, returns 0 on success and 1 on error. */
int negotiate(int fd, struct vif_info *vif)
{
	enum command command = ethernet;
	struct init_message *me;
	struct ethernet_args args;
	struct init_message you;
	char *txt;

	me = create_init_message();
	if (!me)
		goto err;

	if (write_init_message(fd, me) == -1)
		goto err;

	if (read_init_message(fd, &you) == -1)
		goto err;

	if (me->version != you.version) {
		syslog(LOG_CRIT, "Server did not accept our protocol version (client: %d, server: %d)", me->version, you.version);
		goto err;
	}

	txt = print_init_message(&you);
	if (!txt)
		goto err;

	syslog(LOG_INFO, "Server reports %s", txt);
	free(txt);

	if (write_command(fd, &command) == -1)
		goto err;

	/* We don't need a uuid */
	memset(&args.uuid_string[0], 0, sizeof(args.uuid_string));
	if (write_ethernet_args(fd, &args) == -1)
		goto err;

	if (read_vif_response(fd, vif) == -1)
		goto err;

	return 0;
err:
	syslog(LOG_CRIT, "Failed to negotiate vmnet connection");
	return 1;
}

/* Argument passed to proxy threads */
struct connection {
	int fd;              /* Hyper-V socket with vmnet protocol */
	int tapfd;           /* TAP device with ethernet frames */
	struct vif_info vif; /* Contains MAC, MTU etc, received from server */
	struct ring* from_vmnet_ring;
	struct ring* to_vmnet_ring;
	int message_size;    /* Maximum size of a Hyper-V read or write */
};

/* Trim the iovec so that it contains at most len bytes. */
void trim_iovec(struct iovec *iovec, int *iovec_len, size_t len)
{
	for (int i = 0; i < *iovec_len; i++) {
		if (iovec[i].iov_len > len) {
			iovec[i].iov_len = len;
			*iovec_len = i + 1;
			return;
		}
		len -= iovec[i].iov_len;
	}
}

size_t len_iovec(struct iovec *iovec, int iovec_len)
{
	size_t len = 0;
	for (int i = 0; i < iovec_len; i++) {
		len += iovec[i].iov_len;
	}
	return len;
}

/* Read bytes from vmnet into the from_vmnet_ring */
static void* vmnet_to_ring(void *arg)
{
	struct connection *c = (struct connection *)arg;
	struct ring *ring = c->from_vmnet_ring;
	struct iovec iovec[2]; /* We won't need more than 2 for the ring */
	int iovec_len;
	while (1) {
		iovec_len = sizeof(iovec) / sizeof(struct iovec);
		TRC("vmnet_to_ring: ring_producer_wait_available n=%d iovec_len=%d\n", 1, iovec_len);
		if (ring_producer_wait_available(ring, 1, &iovec[0], &iovec_len) != 0) {
			fatal("Failed to read a data from vmnet");
		}
		trim_iovec(iovec, &iovec_len, c->message_size);
		{
			int length = 0;
			for (int i = 0; i < iovec_len; i ++) {
				length += iovec[i].iov_len;
			}
			TRC("vmnet_to_ring readv len %d\n", length);
		}
		ssize_t n = readv(c->fd, &iovec[0], iovec_len);
		TRC("vmnet_to_ring: read %zd\n", n);
		if (n == 0) {
			syslog(LOG_CRIT, "EOF reading from socket: closing\n");
			ring_producer_eof(ring);
			goto err;
		}
		if (n < 0) {
			syslog(LOG_CRIT,
						 "Failure reading from socket: closing: %s (%d)",
						 strerror(errno), errno);
			ring_producer_eof(ring);
			goto err;
		}
		TRC("vmnet_to_ring: advance producer %zd\n", n);
		ring_producer_advance(ring, (size_t) n);
	}
err:
	/*
	 * On error: stop reading from the socket and trigger a clean
	 * shutdown
	 */
	TRC("vmnet_to_ring: shutdown\n");
	shutdown(c->fd, SHUT_RD);
	return NULL;
}

/* Decode packets on the from_vmnet_ring and write to the tap device */
static void* ring_to_tap(void *arg)
{
	struct connection *c = (struct connection *)arg;
	struct iovec iovec[2]; /* We won't need more than 2 for the ring */
	int iovec_len;
	int length;
	struct ring *ring = c->from_vmnet_ring;
	while (1) {
		/* Read the packet length: this requires 2 bytes */
		iovec_len = sizeof(iovec) / sizeof(struct iovec);
		TRC("ring_to_tap: ring_consumer_wait_available n=%d iovec_len=%d\n", 2, iovec_len);
		if (ring_consumer_wait_available(ring, 2, &iovec[0], &iovec_len) != 0) {
			fatal("Failed to read a packet header from host");
		}
		length = *((uint8_t*)iovec[0].iov_base) & 0xff;
		/* The second byte might be in the second iovec array */
		if (iovec[0].iov_len >= 2) {
			length |= (*((uint8_t*)iovec[0].iov_base + 1) & 0xff) << 8;
		} else {
			length |= (*((uint8_t*)iovec[1].iov_base) & 0xff) << 8;
		}
		assert(length > 0);
		TRC("ring_to_tap: packet of length %d\n", length);
		if (length > max_packet_size) {
			syslog(LOG_CRIT,
			       "Received an over-large packet: %d > %ld",
			       length, max_packet_size);
			exit(1);
		}
		ring_consumer_advance(ring, 2);

		/* Read the variable length packet */
		iovec_len = sizeof(iovec) / sizeof(struct iovec);
		TRC("ring_to_tap: ring_consumer_wait_available n=%d iovec_len=%d\n", length, iovec_len);
		if (ring_consumer_wait_available(ring, length, &iovec[0], &iovec_len) != 0) {
			fatal("Failed to read a packet body from host");
		}
		assert(len_iovec(&iovec[0], iovec_len) >= length);
		trim_iovec(iovec, &iovec_len, length);
		ssize_t n = writev(c->tapfd, &iovec[0], iovec_len);
		if (n != length) {
			syslog(LOG_CRIT,
						 "Failed to write %d bytes to tap device (wrote %d)", length, n);
			//exit(1);
		}
		TRC("ring_to_tap: ring_consumer_advance n=%zd\n", n);
		ring_consumer_advance(ring, (size_t) length);
	}
	return NULL;
}

/* Write packets with header from the tap device onto the to_vmnet_ring */
static void *tap_to_ring(void *arg)
{
	struct connection *connection = (struct connection *)arg;
	struct ring *ring = connection->to_vmnet_ring;
	struct iovec iovec[2]; /* We won't need more than 2 for the ring */
	int iovec_len;
	struct iovec payload[2]; /* The packet body after the 2 byte header */
	int payload_len;
	size_t length;
	while (1) {
		/* Wait for space for a 2 byte header + max_packet_size */
		length = 2 + connection->vif.max_packet_size;
		iovec_len = sizeof(iovec) / sizeof(struct iovec);
		TRC("tap_to_ring: ring_producer_wait_available n=%zd iovec_len=%d\n", length, iovec_len);
		if (ring_producer_wait_available(ring, length, &iovec[0], &iovec_len) != 0) {
			fatal("Failed to find enough free space for a packet");
		}
		assert(iovec_len > 0);
		assert(iovec[0].iov_len > 0);
		memcpy(&payload[0], &iovec[0], sizeof(struct iovec) * iovec_len);
		payload_len = iovec_len;

		/* take the first 2 bytes of the free space which will contain the header */
		char *header1 = payload[0].iov_base;
		payload[0].iov_base++;
		payload[0].iov_len--;
		if (payload[0].iov_len == 0) {
			assert(payload_len == 2); /* because `length` > 1 */
			payload[0].iov_base = payload[1].iov_base;
			payload[0].iov_len = payload[1].iov_len;
			payload_len --;
		}
		char *header2 = payload[0].iov_base;
		payload[0].iov_base++;
		payload[0].iov_len--;
		/* payload is now where the packet should go */

		/* limit the message size */
		trim_iovec(payload, &payload_len, connection->message_size);

		length = readv(connection->tapfd, payload, payload_len);

		if (length == -1) {
			if (errno == ENXIO)
				fatal("tap device has gone down");

			syslog(LOG_WARNING, "ignoring error %d", errno);
			/*
			 * This is what mirage-net-unix does. Is it a good
			 * idea really?
			 */
			exit(1);
		}
		*header1 = (length >> 0) & 0xff;
		*header2 = (length >> 8) & 0xff;
		TRC("tap_to_ring: ring_producer_advance n=%zd\n", length + 2);

		ring_producer_advance(ring, (size_t) (length + 2));
	}
	return NULL;
}

/* Write bytes from the to_vmnet_ring to the vmnet fd */
static void *ring_to_vmnet(void *arg)
{
	struct connection *c = (struct connection *)arg;
	struct iovec iovec[2]; /* We won't need more than 2 for the ring */
	int iovec_len;
	int length;
	struct ring *ring = c->to_vmnet_ring;
	while (1) {
		/* Read the packet length: this requires 2 bytes */
		iovec_len = sizeof(iovec) / sizeof(struct iovec);
		TRC("ring_to_vmnet: ring_producer_wait_available n=%d iovec_len=%d\n", 1, iovec_len);
		if (ring_consumer_wait_available(ring, 1, &iovec[0], &iovec_len) != 0) {
			fatal("Failed to read data from ring");
		}
		trim_iovec(iovec, &iovec_len, c->message_size);
		length = 0;
		for (int i = 0; i < iovec_len; i++ ) {
			length += iovec[i].iov_len;
		}
		TRC("ring_to_vmnet: read %d bytes\n", length);
		ssize_t n = writev(c->fd, &iovec[0], iovec_len);

		TRC("ring_to_vmnet: advance consumer %zd\n", n);
		ring_consumer_advance(ring, (size_t) n);
	}
	return NULL;
}

/*
 * Handle a connection by exchanging ethernet frames forever.
 */
static void handle(struct connection *connection)
{
	pthread_t v2r, r2t, t2r, r2v;

	if (pthread_create(&t2r, NULL, tap_to_ring, connection) != 0)
		fatal("Failed to create the tap_to_ring thread");

	if (pthread_create(&v2r, NULL, vmnet_to_ring, connection) != 0)
		fatal("Failed to create the vmnet_to_tap thread");

	if (pthread_create(&r2t, NULL, ring_to_tap, connection) != 0)
		fatal("Failed to create the ring_to_tap thread");

	if (pthread_create(&r2v, NULL, ring_to_vmnet, connection) != 0)
		fatal("Failed to create the ring_to_vmnet thread");

	if (pthread_join(t2r, NULL) != 0)
		fatal("Failed to join the tap_to_ring thread");

	if (pthread_join(v2r, NULL) != 0)
		fatal("Failed to join the vmnet_to_ring thread");

	if (pthread_join(t2r, NULL) != 0)
		fatal("Failed to join the tap_to_ring thread");

	if (pthread_join(r2v, NULL) != 0)
		fatal("Failed to join the ring_to_vmnet thread");
}

static int create_listening_socket(GUID serviceid)
{
	SOCKADDR_HV sa;
	int lsock;
	int res;

	lsock = socket(AF_HYPERV, SOCK_STREAM, HV_PROTOCOL_RAW);
	if (lsock == -1)
		fatal("socket()");

	sa.Family = AF_HYPERV;
	sa.Reserved = 0;
	sa.VmId = HV_GUID_WILDCARD;
	sa.ServiceId = serviceid;

	res = bind(lsock, (const struct sockaddr *)&sa, sizeof(sa));
	if (res == -1)
		fatal("bind()");

	res = listen(lsock, SOMAXCONN);
	if (res == -1)
		fatal("listen()");

	return lsock;
}

static int connect_socket(GUID serviceid)
{
	SOCKADDR_HV sa;
	int sock;
	int res;

	sock = socket(AF_HYPERV, SOCK_STREAM, HV_PROTOCOL_RAW);
	if (sock == -1)
		fatal("socket()");

	sa.Family = AF_HYPERV;
	sa.Reserved = 0;
	sa.VmId = HV_GUID_PARENT;
	sa.ServiceId = serviceid;

	res = connect(sock, (const struct sockaddr *)&sa, sizeof(sa));
	if (res == -1)
		fatal("connect()");

	return sock;
}

static int accept_socket(int lsock)
{
	SOCKADDR_HV sac;
	socklen_t socklen = sizeof(sac);
	int csock;

	csock = accept(lsock, (struct sockaddr *)&sac, &socklen);
	if (csock == -1)
		fatal("accept()");

	syslog(LOG_INFO, "Connect from: " GUID_FMT ":" GUID_FMT "\n",
	       GUID_ARGS(sac.VmId), GUID_ARGS(sac.ServiceId));

	return csock;
}

void write_pidfile(const char *pidfile)
{
	pid_t pid = getpid();
	char *pid_s;
	FILE *file;
	int len;

	if (asprintf(&pid_s, "%lld", (long long)pid) == -1)
		fatal("Failed to allocate pidfile string");

	len = strlen(pid_s);
	file = fopen(pidfile, "w");
	if (file == NULL) {
		syslog(LOG_CRIT, "Failed to open pidfile %s", pidfile);
		exit(1);
	}

	if (fwrite(pid_s, 1, len, file) != len)
		fatal("Failed to write pid to pidfile");

	fclose(file);
	free(pid_s);
}

void daemonize(const char *pidfile)
{
	pid_t pid;
	int null;

	pid = fork();
	if (pid == -1)
		fatal("Failed to fork()");
	else if (pid != 0)
		exit(0);

	if (setsid() == -1)
		fatal("Failed to setsid()");

	if (chdir("/") == -1)
		fatal("Failed to chdir()");

	null = open("/dev/null", O_RDWR);
	if (null == -1)
		fatal("Failed to open /dev/null");
	dup2(null, STDIN_FILENO);
	dup2(null, STDOUT_FILENO);
	dup2(null, STDERR_FILENO);
	close(null);

	if (pidfile)
		write_pidfile(pidfile);
}

void usage(char *name)
{
	printf("%s usage:\n", name);
	printf("\t[--daemon] [--tap <name>] [--serviceid <guid>] [--pid <file>]\n");
	printf("\t[--message-size <bytes>] [--buffer-size <bytes>]\n");
	printf("\t[--listen | --connect]\n\n");
	printf("where\n");
	printf("\t--daemonize: run as a background daemon\n");
	printf("\t--nofork: don't run handlers in subprocesses\n");
	printf("\t--tap <name>: create a tap device with the given name\n");
	printf("\t  (defaults to eth1)\n");
	printf("\t--serviceid <guid>: use <guid> as the well-known service GUID\n");
	printf("\t  (defaults to %s)\n", default_sid);
	printf("\t--pid <file>: write a pid to the given file\n");
	printf("\t--message-size <bytes>: dictates the maximum transfer size for AF_HVSOCK\n");
	printf("\t--buffer-size <bytes>: dictates the buffer size for AF_HVSOCK\n");
	printf("\t--listen: listen forever for incoming AF_HVSOCK connections\n");
	printf("\t--connect: connect to the parent partition\n");
}

int main(int argc, char **argv)
{
	char *serviceid = default_sid;
	struct connection connection;
	char *tap = "eth1";
	char *pidfile = NULL;
	int lsocket = -1;
	int sock = -1;
	int res = 0;
	int status;
	pid_t child;
	int tapfd;
	int ring_size = 1048576;
	int message_size = 8192; /* Well known to work across Hyper-V versions */
	GUID sid;
	int c;

	int option_index;
	int log_flags = LOG_CONS | LOG_NDELAY;
	static struct option long_options[] = {
		/* These options set a flag. */
		{"daemon", no_argument, &daemon_flag, 1},
		{"nofork", no_argument, &nofork_flag, 1},
		{"serviceid", required_argument, NULL, 's'},
		{"tap", required_argument, NULL, 't'},
		{"pidfile", required_argument, NULL, 'p'},
		{"listen", no_argument, &listen_flag, 1},
		{"connect", no_argument, &connect_flag, 1},
		{"buffer-size", required_argument, NULL, 'b'},
		{"message-size", required_argument, NULL, 'm'},
		{0, 0, 0, 0}
	};

	opterr = 0;
	while (1) {
		option_index = 0;

		c = getopt_long(argc, argv, "ds:t:p:r:m:v",
				long_options, &option_index);
		if (c == -1)
			break;

		switch (c) {
		case 'd':
			daemon_flag = 1;
			break;
		case 'n':
			nofork_flag = 1;
			break;
		case 's':
			serviceid = optarg;
			break;
		case 't':
			tap = optarg;
			break;
		case 'p':
			pidfile = optarg;
			break;
		case 'b':
			ring_size = atoi(optarg);
			break;
		case 'm':
			message_size = atoi(optarg);
			break;
		case 'v':
			verbose ++;
			break;
		case 0:
			break;
		default:
			usage(argv[0]);
			exit(1);
		}
	}

	if ((listen_flag && connect_flag) || !(listen_flag || connect_flag)) {
		fprintf(stderr, "Please supply either the --listen or --connect flag, but not both.\n");
		exit(1);
	}

	if (daemon_flag && !pidfile) {
		fprintf(stderr, "For daemon mode, please supply a --pidfile argument.\n");
		exit(1);
	}

	res = parseguid(serviceid, &sid);
	if (res) {
		fprintf(stderr, "Failed to parse serviceid as GUID: %s\n", serviceid);
		usage(argv[0]);
		exit(1);
	}

	if (!daemon_flag)
		log_flags |= LOG_PERROR;

	openlog(argv[0], log_flags, LOG_DAEMON);

	tapfd = alloc_tap(tap);
	connection.tapfd = tapfd;
	connection.to_vmnet_ring = ring_allocate(ring_size);
	connection.from_vmnet_ring = ring_allocate(ring_size);
	connection.message_size = message_size;
	if (listen_flag) {
		syslog(LOG_INFO, "starting in listening mode with serviceid=%s and tap=%s", serviceid, tap);
		lsocket = create_listening_socket(sid);
	} else {
		syslog(LOG_INFO, "starting in connect mode with serviceid=%s and tap=%s", serviceid, tap);
	}

	for (;;) {
		if (sock != -1) {
			close(sock);
			sock = -1;
		}

		if (listen_flag)
			sock = accept_socket(lsocket);
		else
			sock = connect_socket(sid);

		connection.fd = sock;
		if (negotiate(sock, &connection.vif) != 0) {
			sleep(1);
			continue;
		}

		syslog(LOG_INFO, "VMNET VIF has MAC %02x:%02x:%02x:%02x:%02x:%02x",
		       connection.vif.mac[0], connection.vif.mac[1], connection.vif.mac[2],
		       connection.vif.mac[3], connection.vif.mac[4], connection.vif.mac[5]
			);
		set_macaddr(tap, &connection.vif.mac[0]);
		set_mtu(tap, connection.vif.mtu);

		/* Daemonize after we've made our first reliable connection */
		if (daemon_flag) {
			daemon_flag = 0;
			daemonize(pidfile);
		}
		if (nofork_flag) {
			handle(&connection);
			exit(1);
		}
		/*
		 * Run the multithreaded part in a subprocess. On error the
		 * process will exit() which tears down all the threads
		 */
		child = fork();
		if (child == 0) {
			handle(&connection);
			/*
			 * should never happen but just in case of a logic
			 * bug in handle
			 */
			exit(1);
		}

		for (;;) {
			if (waitpid(child, &status, 0) != -1)
				break;
		}
	}
}
