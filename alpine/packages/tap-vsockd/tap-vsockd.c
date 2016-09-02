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
#include <pthread.h>
#include <arpa/inet.h>
#include <sys/ioctl.h>
#include <net/if.h>
#include <linux/if_tun.h>
#include <net/if_arp.h>

#include <sys/types.h>
#include <sys/socket.h>
#include <sys/wait.h>
#include <ifaddrs.h>

#include "hvsock.h"
#include "protocol.h"

int daemon_flag;
int listen_flag;
int connect_flag;

char *default_sid = "30D48B34-7D27-4B0B-AAAF-BBBED334DD59";

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
	strcpy(ifq.ifr_name, dev);
	memcpy(&ifq.ifr_hwaddr.sa_data[0], mac, 6);
	ifq.ifr_hwaddr.sa_family = ARPHRD_ETHER;

	if (ioctl(fd, SIOCSIFHWADDR, &ifq) == -1)
		fatal("SIOCSIFHWADDR failed");

	close(fd);
}

/* Negotiate a vmnet connection, returns 0 on success and 1 on error. */
int negotiate(int fd, struct vif_info *vif)
{
	/* Negotiate with com.docker.slirp */
	struct init_message *me = create_init_message();
	enum command command = ethernet;
	struct ethernet_args args;
	struct init_message you;
	char *txt;

	if (write_init_message(fd, me) == -1)
		goto err;

	if (read_init_message(fd, &you) == -1)
		goto err;

	txt = print_init_message(&you);
	syslog(LOG_INFO, "Server reports %s", txt);
	free(txt);

	if (write_command(fd, &command) == -1)
		goto err;

	/* We don't need a uuid */
	memset(&args.uuid_string[0], 0, sizeof(args.uuid_string));
	if (write_ethernet_args(fd, &args) == -1)
		goto err;

	if (read_vif_info(fd, vif) == -1)
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
};

static void *vmnet_to_tap(void *arg)
{
	struct connection *connection = (struct connection *)arg;
	uint8_t buffer[2048];
	uint8_t header[2];
	int length, n;

	for (;;) {
		if (really_read(connection->fd, &header[0], 2) == -1)
			fatal("Failed to read a packet header from host");

		length = (header[0] & 0xff) | ((header[1] & 0xff) << 8);
		if (length > sizeof(buffer)) {
			syslog(LOG_CRIT,
			       "Received an over-large packet: %d > %ld",
			       length, sizeof(buffer));
			exit(1);
		}

		if (really_read(connection->fd, &buffer[0], length) == -1) {
			syslog(LOG_CRIT,
			       "Failed to read packet contents from host");
			exit(1);
		}

		n = write(connection->tapfd, &buffer[0], length);
		if (n != length) {
			syslog(LOG_CRIT,
			       "Failed to write %d bytes to tap device (wrote %d)", length, n);
			exit(1);
		}
	}
}

static void *tap_to_vmnet(void *arg)
{
	struct connection *connection = (struct connection *)arg;
	uint8_t buffer[2048];
	uint8_t header[2];
	int length;

	for (;;) {
		length = read(connection->tapfd, &buffer[0], sizeof(buffer));
		if (length == -1) {
			if (errno == ENXIO)
				fatal("tap device has gone down");

			syslog(LOG_WARNING, "ignoring error %d", errno);
			/*
			 * This is what mirage-net-unix does. Is it a good
			 * idea really?
			 */
			continue;
		}

		header[0] = (length >> 0) & 0xff;
		header[1] = (length >> 8) & 0xff;
		if (really_write(connection->fd, &header[0], 2) == -1)
			fatal("Failed to write packet header");

		if (really_write(connection->fd, &buffer[0], length) == -1)
			fatal("Failed to write packet body");
	}

	return NULL;
}

/*
 * Handle a connection by exchanging ethernet frames forever.
 */
static void handle(struct connection *connection)
{
	pthread_t v2t, t2v;

	if (pthread_create(&v2t, NULL, vmnet_to_tap, connection) != 0)
		fatal("Failed to create the vmnet_to_tap thread");

	if (pthread_create(&t2v, NULL, tap_to_vmnet, connection) != 0)
		fatal("Failed to create the tap_to_vmnet thread");

	if (pthread_join(v2t, NULL) != 0)
		fatal("Failed to join the vmnet_to_tap thread");

	if (pthread_join(t2v, NULL) != 0)
		fatal("Failed to join the tap_to_vmnet thread");
}

static int create_listening_socket(GUID serviceid)
{
	SOCKADDR_HV sa;
	int lsock = -1;
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
	int sock = -1;
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
	int csock = -1;

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
	printf("\t[--listen | --connect]\n\n");
	printf("where\n");
	printf("\t--daemonize: run as a background daemon\n");
	printf("\t--tap <name>: create a tap device with the given name\n");
	printf("\t  (defaults to eth1)\n");
	printf("\t--serviceid <guid>: use <guid> as the well-known service GUID\n");
	printf("\t  (defaults to %s)\n", default_sid);
	printf("\t--pid <file>: write a pid to the given file\n");
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
	GUID sid;
	int c;

	int option_index;
	int log_flags = LOG_CONS | LOG_NDELAY;
	static struct option long_options[] = {
		/* These options set a flag. */
		{"daemon", no_argument, &daemon_flag, 1},
		{"serviceid", required_argument, NULL, 's'},
		{"tap", required_argument, NULL, 't'},
		{"pidfile", required_argument, NULL, 'p'},
		{"listen", no_argument, &listen_flag, 1},
		{"connect", no_argument, &connect_flag, 1},
		{0, 0, 0, 0}
	};

	opterr = 0;
	while (1) {
		option_index = 0;

		c = getopt_long(argc, argv, "ds:t:p:",
				long_options, &option_index);
		if (c == -1)
			break;

		switch (c) {
		case 'd':
			daemon_flag = 1;
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

		/* Daemonize after we've made our first reliable connection */
		if (daemon_flag) {
			daemon_flag = 0;
			daemonize(pidfile);
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
