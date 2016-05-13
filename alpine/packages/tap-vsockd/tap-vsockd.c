/*
 */

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

int verbose_flag = 0;

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
  fprintf(stderr, "successfully created TAP device %s\n", dev);
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
    fprintf(stderr, "%s Error: %d. %s", msg, errno, strerror(errno));
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
    fprintf(stderr, "Server reports %s\n", txt);
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
    fprintf(stderr, "Failed to negotiate with com.docker.slirp\n");
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
      fprintf(stderr, "Failed to read a packet header from host\n");
      exit(1);
    }
    length = (header[0] & 0xff) | ((header[1] & 0xff) << 8);
    if (length > sizeof(buffer)) {
      fprintf(stderr, "Received an over-large packet: %d > %ld\n", length, sizeof(buffer));
      exit(1);
    }
    if (really_read(connection->fd, &buffer[0], length) == -1){
      fprintf(stderr, "Failed to read packet contents from host\n");
      exit(1);
    }
    n = write(connection->tapfd, &buffer[0], length);
    if (n != length) {
      fprintf(stderr, "Failed to write %d bytes to tap device (wrote %d)\n", length, n);
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
        fprintf(stderr, "tap device has gone down\n");
        exit(0);
      }
      fprintf(stderr, "ignoring error %d\n", errno);
      /* This is what mirage-net-unix does. Is it a good idea really? */
      continue;
    }
    header[0] = (length >> 0) & 0xff;
    header[1] = (length >> 8) & 0xff;
    if (really_write(connection->fd, &header[0], 2) == -1){
      fprintf(stderr, "Failed to write packet header\n");
      exit(1);
    }
    if (really_write(connection->fd, &buffer[0], length) == -1) {
      fprintf(stderr, "Failed to write packet body\n");
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
    fprintf(stderr, "VMNET VIF has MAC %02x:%02x:%02x:%02x:%02x:%02x\n",
      connection.vif.mac[0], connection.vif.mac[1], connection.vif.mac[2],
      connection.vif.mac[3], connection.vif.mac[4], connection.vif.mac[5]
    );

    int tapfd = alloc_tap(tap);
    set_macaddr(tap, &connection.vif.mac[0]);
    connection.tapfd = tapfd;

    if (pthread_create(&v2t, NULL, vmnet_to_tap, &connection) != 0){
      fprintf(stderr, "Failed to create the vmnet_to_tap thread\n");
      exit(1);
    }
    if (pthread_create(&t2v, NULL, tap_to_vmnet, &connection) != 0){
      fprintf(stderr, "Failed to create the tap_to_vmnet thread\n");
      exit(1);
    }
    if (pthread_join(v2t, NULL) != 0){
      fprintf(stderr, "Failed to join the vmnet_to_tap thread\n");
      exit(1);
    }
    if (pthread_join(t2v, NULL) != 0){
      fprintf(stderr, "Failed to join the tap_to_vmnet thread\n");
      exit(1);
    }
}


/* Server:
 * accept() in an endless loop, handle a connection at a time
 */
static int server(GUID serviceid, const char *tap)
{
    SOCKET lsock = INVALID_SOCKET;
    SOCKET csock = INVALID_SOCKET;
    SOCKADDR_HV sa, sac;
    socklen_t socklen = sizeof(sac);
    int res;

    lsock = socket(AF_HYPERV, SOCK_STREAM, HV_PROTOCOL_RAW);
    if (lsock == INVALID_SOCKET) {
        sockerr("socket()");
        return 1;
    }

    sa.Family = AF_HYPERV;
    sa.Reserved = 0;
    sa.VmId = HV_GUID_WILDCARD;
    sa.ServiceId = serviceid;

    res = bind(lsock, (const struct sockaddr *)&sa, sizeof(sa));
    if (res == SOCKET_ERROR) {
        sockerr("bind()");
        closesocket(lsock);
        return 1;
    }

    res = listen(lsock, SOMAXCONN);
    if (res == SOCKET_ERROR) {
        sockerr("listen()");
        closesocket(lsock);
        return 1;
    }

    while(1) {
        csock = accept(lsock, (struct sockaddr *)&sac, &socklen);
        if (csock == INVALID_SOCKET) {
            sockerr("accept()");
            closesocket(lsock);
            return 1;
        }

        printf("Connect from: "GUID_FMT":"GUID_FMT"\n",
               GUID_ARGS(sac.VmId), GUID_ARGS(sac.ServiceId));

        handle(csock, tap);
        closesocket(csock);
    }
}

void usage(char *name)
{
    printf("%s: --verbose | --service id <id>  | --tap <tap>\n", name);
    printf("<id>: Hyper-V socket serviceId to bind\n");
    printf("<tap>: tap device to connect to\n");
}

int __cdecl main(int argc, char **argv)
{
    int res = 0;
    GUID sid;
    int c;
    /* Defaults to a testing GUID */
    char *serviceid = "3049197C-9A4E-4FBF-9367-97F792F16994";
    char *tap = "eth1";

    opterr = 0;
    while (1) {
      static struct option long_options[] = {
        /* These options set a flag. */
        {"verbose",   no_argument,       &verbose_flag, 1},
        {"serviceid", required_argument, NULL, 's'},
        {"tap",       required_argument, NULL, 't'},
        {0, 0, 0, 0}
      };
      int option_index = 0;

      c = getopt_long (argc, argv, "vs:t:", long_options, &option_index);
      if (c == -1) break;

      switch (c) {
        case 'v':
          verbose_flag = 1;
          break;
        case 's':
          serviceid = optarg;
          break;
        case 't':
          tap = optarg;
          break;
        case 0:
          break;
        default:
          usage (argv[0]);
          exit (1);
      }
    }
    fprintf(stderr, "serviceid=%s\n", serviceid);
    fprintf(stderr, "tap=%s\n", tap);
    res = parseguid(serviceid, &sid);
    if (res) {
      fprintf(stderr, "Failed to parse serviceid as GUID: %s\n", serviceid);
      usage(argv[0]);
      exit(1);
    }

    server(sid, tap);

    return 0;
}
