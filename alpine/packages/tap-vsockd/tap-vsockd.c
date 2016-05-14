/*
 */

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
#include <arpa/inet.h>
#include <sys/ioctl.h>
#include <net/if.h>
#include <linux/if_tun.h>
#include <net/if_arp.h>

#include <sys/types.h>
#include <sys/socket.h>
#include <ifaddrs.h>


#include "compat.h"
#include "protocol.h"

int debug_flag = 0;
int listen_flag = 0;
int connect_flag = 0;

int alloc_tap(const char *dev) {
  int fd;
  struct ifreq ifr;
  const char *clonedev = "/dev/net/tun";
  if ((fd = open(clonedev, O_RDWR)) == -1) {
    perror("Failed to open /dev/net/tun");
    exit(1);
  }
  memset(&ifr, 0, sizeof(ifr));
  ifr.ifr_flags = IFF_TAP | IFF_NO_PI;
  strncpy(ifr.ifr_name, dev, IFNAMSIZ);
  if (ioctl(fd, TUNSETIFF, (void*) &ifr) < 0) {
    perror("TUNSETIFF failed");
    exit(1);
  }
  int persist = 1;
  if (ioctl(fd, TUNSETPERSIST, persist) < 0) {
    perror("TUNSETPERSIST failed");
    exit(1);
  }
  syslog(LOG_INFO, "successfully created TAP device %s", dev);
  return fd;
}

void set_macaddr(const char *dev, uint8_t *mac) {
  int fd;
  struct ifreq ifq;

  fd = socket(PF_INET, SOCK_DGRAM, 0);
  strcpy(ifq.ifr_name, dev);
  memcpy(&ifq.ifr_hwaddr.sa_data[0], mac, 6);
  ifq.ifr_hwaddr.sa_family = ARPHRD_ETHER;

  if (ioctl(fd, SIOCSIFHWADDR, &ifq) == -1) {
    perror("SIOCSIFHWADDR failed");
    exit(1);
  }

  close(fd);
}

#define SVR_BUF_LEN (3 * 4096)
#define MAX_BUF_LEN (2 * 1024 * 1024)


/* Helper macros for parsing/printing GUIDs */
#define GUID_FMT "%08x-%04hx-%04hx-%02x%02x-%02x%02x%02x%02x%02x%02x"
#define GUID_ARGS(_g)                                               \
    (_g).Data1, (_g).Data2, (_g).Data3,                             \
    (_g).Data4[0], (_g).Data4[1], (_g).Data4[2], (_g).Data4[3],     \
    (_g).Data4[4], (_g).Data4[5], (_g).Data4[6], (_g).Data4[7]
#define GUID_SARGS(_g)                                              \
    &(_g).Data1, &(_g).Data2, &(_g).Data3,                          \
    &(_g).Data4[0], &(_g).Data4[1], &(_g).Data4[2], &(_g).Data4[3], \
    &(_g).Data4[4], &(_g).Data4[5], &(_g).Data4[6], &(_g).Data4[7]


int parseguid(const char *s, GUID *g)
{
    int res;
    int p0, p1, p2, p3, p4, p5, p6, p7;

    res = sscanf(s, GUID_FMT,
                 &g->Data1, &g->Data2, &g->Data3,
                 &p0, &p1, &p2, &p3, &p4, &p5, &p6, &p7);
    if (res != 11)
        return 1;
    g->Data4[0] = p0;
    g->Data4[1] = p1;
    g->Data4[2] = p2;
    g->Data4[3] = p3;
    g->Data4[4] = p4;
    g->Data4[5] = p5;
    g->Data4[6] = p6;
    g->Data4[7] = p7;
    return 0;
}

/* Slightly different error handling between Windows and Linux */
void sockerr(const char *msg)
{
    syslog(LOG_CRIT, "%s Error: %d. %s", msg, errno, strerror(errno));
}

void negotiate(SOCKET fd, struct vif_info *vif)
{
    /* Negotiate with com.docker.slirp */
    struct init_message *me = create_init_message();
    if (write_init_message(fd, me) == -1) {
      goto err;
    }
    struct init_message you;
    if (read_init_message(fd, &you) == -1) {
      goto err;
    }
    char *txt = print_init_message(&you);
    syslog(LOG_INFO, "Server reports %s", txt);
    free(txt);
    enum command command = ethernet;
    if (write_command(fd, &command) == -1) {
      goto err;
    }
    struct ethernet_args args;
    /* We don't need a uuid */
    memset(&args.uuid_string[0], 0, sizeof(args.uuid_string));
    if (write_ethernet_args(fd, &args) == -1) {
      goto err;
    }
    if (read_vif_info(fd, vif) == -1) {
      goto err;
    }
    return;
err:
    syslog(LOG_CRIT, "Failed to negotiate with com.docker.slirp");
    exit(1);
}


/* Argument passed to proxy threads */
struct connection {
    SOCKET fd; /* Hyper-V socket with vmnet protocol */
    int tapfd; /* TAP device with ethernet frames */
    struct vif_info vif; /* Contains VIF MAC, MTU etc, received from server */
};

static void* vmnet_to_tap(void *arg)
{
  int length, n;
  struct connection *connection = (struct connection*) arg;
  uint8_t header[2];
  uint8_t buffer[2048];

  for (;;) {
    if (really_read(connection->fd, &header[0], 2) == -1){
      syslog(LOG_CRIT, "Failed to read a packet header from host");
      exit(1);
    }
    length = (header[0] & 0xff) | ((header[1] & 0xff) << 8);
    if (length > sizeof(buffer)) {
      syslog(LOG_CRIT, "Received an over-large packet: %d > %ld", length, sizeof(buffer));
      exit(1);
    }
    if (really_read(connection->fd, &buffer[0], length) == -1){
      syslog(LOG_CRIT, "Failed to read packet contents from host");
      exit(1);
    }
    n = write(connection->tapfd, &buffer[0], length);
    if (n != length) {
      syslog(LOG_CRIT, "Failed to write %d bytes to tap device (wrote %d)", length, n);
      exit(1);
    }
  }
}

static void* tap_to_vmnet(void *arg)
{
  int length;
  struct connection *connection = (struct connection*) arg;
  uint8_t header[2];
  uint8_t buffer[2048];

  for (;;) {
    length = read(connection->tapfd, &buffer[0], sizeof(buffer));
    if (length == -1) {
      if (errno == ENXIO) {
        syslog(LOG_CRIT, "tap device has gone down");
        exit(0);
      }
      syslog(LOG_WARNING, "ignoring error %d", errno);
      /* This is what mirage-net-unix does. Is it a good idea really? */
      continue;
    }
    header[0] = (length >> 0) & 0xff;
    header[1] = (length >> 8) & 0xff;
    if (really_write(connection->fd, &header[0], 2) == -1){
      syslog(LOG_CRIT, "Failed to write packet header");
      exit(1);
    }
    if (really_write(connection->fd, &buffer[0], length) == -1) {
      syslog(LOG_CRIT, "Failed to write packet body");
      exit(1);
    }
  }
}

/* Handle a connection. Handshake with the com.docker.slirp process and start
 * exchanging ethernet frames between the socket and the tap device.
 */
static void handle(SOCKET fd, const char *tap)
{
    struct connection connection;
    pthread_t v2t, t2v;

    connection.fd = fd;
    negotiate(fd, &connection.vif);
    syslog(LOG_INFO, "VMNET VIF has MAC %02x:%02x:%02x:%02x:%02x:%02x",
      connection.vif.mac[0], connection.vif.mac[1], connection.vif.mac[2],
      connection.vif.mac[3], connection.vif.mac[4], connection.vif.mac[5]
    );

    int tapfd = alloc_tap(tap);
    set_macaddr(tap, &connection.vif.mac[0]);
    connection.tapfd = tapfd;

    if (pthread_create(&v2t, NULL, vmnet_to_tap, &connection) != 0){
      syslog(LOG_CRIT, "Failed to create the vmnet_to_tap thread");
      exit(1);
    }
    if (pthread_create(&t2v, NULL, tap_to_vmnet, &connection) != 0){
      syslog(LOG_CRIT, "Failed to create the tap_to_vmnet thread");
      exit(1);
    }
    if (pthread_join(v2t, NULL) != 0){
      syslog(LOG_CRIT, "Failed to join the vmnet_to_tap thread");
      exit(1);
    }
    if (pthread_join(t2v, NULL) != 0){
      syslog(LOG_CRIT, "Failed to join the tap_to_vmnet thread");
      exit(1);
    }
}

static int create_listening_socket(GUID serviceid) {
  SOCKET lsock = INVALID_SOCKET;
  SOCKADDR_HV sa;
  int res;

  lsock = socket(AF_HYPERV, SOCK_STREAM, HV_PROTOCOL_RAW);
  if (lsock == INVALID_SOCKET) {
      sockerr("socket()");
      exit(1);
  }

  sa.Family = AF_HYPERV;
  sa.Reserved = 0;
  sa.VmId = HV_GUID_WILDCARD;
  sa.ServiceId = serviceid;

  res = bind(lsock, (const struct sockaddr *)&sa, sizeof(sa));
  if (res == SOCKET_ERROR) {
      sockerr("bind()");
      closesocket(lsock);
      exit(1);
  }

  res = listen(lsock, SOMAXCONN);
  if (res == SOCKET_ERROR) {
      sockerr("listen()");
      closesocket(lsock);
      exit(1);
  }
  return lsock;
}

static int connect_socket(GUID serviceid) {
  SOCKET sock = INVALID_SOCKET;
  SOCKADDR_HV sa;
  int res;

  sock = socket(AF_HYPERV, SOCK_STREAM, HV_PROTOCOL_RAW);
  if (sock == INVALID_SOCKET) {
      sockerr("socket()");
      exit(1);
  }

  sa.Family = AF_HYPERV;
  sa.Reserved = 0;
  sa.VmId = HV_GUID_PARENT;
  sa.ServiceId = serviceid;

  res = connect(sock, (const struct sockaddr *)&sa, sizeof(sa));
  if (res == SOCKET_ERROR) {
      sockerr("connect()");
      closesocket(sock);
      exit(1);
  }

  return sock;
}


/* Server:
 * accept() in an endless loop, handle a connection at a time
 */
static void accept_forever(SOCKET lsock, const char *tap)
{
  SOCKET csock = INVALID_SOCKET;
  SOCKADDR_HV sac;
  socklen_t socklen = sizeof(sac);

    while(1) {
        csock = accept(lsock, (struct sockaddr *)&sac, &socklen);
        if (csock == INVALID_SOCKET) {
            sockerr("accept()");
            closesocket(lsock);
            exit(1);
        }

        printf("Connect from: "GUID_FMT":"GUID_FMT"\n",
               GUID_ARGS(sac.VmId), GUID_ARGS(sac.ServiceId));

        handle(csock, tap);
        closesocket(csock);
    }
}

void write_pidfile(const char *pidfile) {
  pid_t pid = getpid();
  char * pid_s;
  FILE *file;
  int len;

  if (asprintf(&pid_s, "%lld", (long long) pid) == -1) {
    syslog(LOG_CRIT, "Failed to allocate pidfile string");
    exit(1);
  }
  len = strlen(pid_s);
  file = fopen(pidfile, "w");
  if (file == NULL) {
    syslog(LOG_CRIT, "Failed to open pidfile %s", pidfile);
    exit(1);
  }

  if (fwrite(pid_s, 1, len, file) != len) {
    syslog(LOG_CRIT, "Failed to write pid to pidfile");
    exit(1);
  }
  fclose(file);
  free(pid_s);
}

void usage(char *name)
{
    printf("%s: [--debug] [--tap <name>] [--serviceid <guid>] [--pid <file>]\n", name);
    printf("\t[--listen | --connect]\n\n");
    printf("--debug: log to stderr as well as syslog\n");
    printf("--tap <name>: create a tap device with the given name (defaults to eth1)\n");
    printf("--serviceid <guid>: use <guid> as the well-known service GUID\n");
    printf("--pid <file>: write a pid to the given file\n");
    printf("--listen: listen forever for incoming AF_HVSOCK connections\n");
    printf("--connect: connect to the parent partition\n");
}

int __cdecl main(int argc, char **argv)
{
    int res = 0;
    GUID sid;
    int c;
    /* Defaults to a testing GUID */
    char *serviceid = "3049197C-9A4E-4FBF-9367-97F792F16994";
    char *tap = "eth1";
    char *pidfile = NULL;

    opterr = 0;
    while (1) {
      static struct option long_options[] = {
        /* These options set a flag. */
        {"debug",     no_argument,       &debug_flag, 1},
        {"serviceid", required_argument, NULL, 's'},
        {"tap",       required_argument, NULL, 't'},
        {"pidfile",   required_argument, NULL, 'p'},
        {"listen",    no_argument,       &listen_flag, 1},
        {"connect",   no_argument,       &connect_flag, 1},
        {0, 0, 0, 0}
      };
      int option_index = 0;

      c = getopt_long (argc, argv, "ds:t:p:", long_options, &option_index);
      if (c == -1) break;

      switch (c) {
        case 'd':
          debug_flag = 1;
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
          usage (argv[0]);
          exit (1);
      }
    }
    if ((listen_flag && connect_flag) || !(listen_flag || connect_flag)){
      fprintf(stderr, "Please supply either the --listen or --connect flag, but not both.\n");
      exit(1);
    }
    int log_flags = LOG_CONS | LOG_NDELAY;
    if (debug_flag) {
      log_flags |= LOG_PERROR;
    }
    openlog(argv[0], log_flags, LOG_DAEMON);

    res = parseguid(serviceid, &sid);
    if (res) {
      syslog(LOG_CRIT, "Failed to parse serviceid as GUID: %s", serviceid);
      usage(argv[0]);
      exit(1);
    }

    if (listen_flag) {
      syslog(LOG_INFO, "starting in listening mode with serviceid=%s and tap=%s", serviceid, tap);
      int socket = create_listening_socket(sid);
      accept_forever(socket, tap);
      exit(0);
    }
    syslog(LOG_INFO, "starting in connect mode with serviceid=%s and tap=%s", serviceid, tap);
    int socket = connect_socket(sid);
    handle(socket, tap);
    return 0;
}
