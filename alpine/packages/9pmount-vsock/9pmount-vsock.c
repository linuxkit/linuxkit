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

#include "compat.h"

int listen_flag = 0;
int connect_flag = 0;

char *default_sid = "C378280D-DA14-42C8-A24E-0DE92A1028E2";

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

void sockerr(const char *msg)
{
    syslog(LOG_CRIT, "%s Error: %d. %s", msg, errno, strerror(errno));
}

static void handle(SOCKET fd)
{
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

static int accept_socket(SOCKET lsock) {
  SOCKET csock = INVALID_SOCKET;
  SOCKADDR_HV sac;
  socklen_t socklen = sizeof(sac);

  csock = accept(lsock, (struct sockaddr *)&sac, &socklen);
  if (csock == INVALID_SOCKET) {
    sockerr("accept()");
    closesocket(lsock);
    exit(1);
  }

  printf("Connect from: "GUID_FMT":"GUID_FMT"\n",
    GUID_ARGS(sac.VmId), GUID_ARGS(sac.ServiceId));
  return csock;
}

void usage(char *name)
{
    printf("%s usage:\n", name);
    printf("\t[--serviceid <guid>] [--listen | --connect]\n\n");
    printf("where\n");
    printf("\t--serviceid <guid>: use <guid> as the well-known service GUID\n");
    printf("\t  (defaults to %s)\n", default_sid);
    printf("\t--listen: listen forever for incoming AF_HVSOCK connections\n");
    printf("\t--connect: connect to the parent partition\n");
}

int __cdecl main(int argc, char **argv)
{
    int res = 0;
    GUID sid;
    int c;
    /* Defaults to a testing GUID */
    char *serviceid = default_sid;

    opterr = 0;
    while (1) {
      static struct option long_options[] = {
        /* These options set a flag. */
        {"serviceid", required_argument, NULL, 's'},
        {"listen",    no_argument,       &listen_flag, 1},
        {"connect",   no_argument,       &connect_flag, 1},
        {0, 0, 0, 0}
      };
      int option_index = 0;

      c = getopt_long (argc, argv, "s:", long_options, &option_index);
      if (c == -1) break;

      switch (c) {

        case 's':
          serviceid = optarg;
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

    int log_flags = LOG_CONS | LOG_NDELAY | LOG_PERROR;

    openlog(argv[0], log_flags, LOG_DAEMON);

    res = parseguid(serviceid, &sid);
    if (res) {
      syslog(LOG_CRIT, "Failed to parse serviceid as GUID: %s", serviceid);
      usage(argv[0]);
      exit(1);
    }

    SOCKET sock = INVALID_SOCKET;
    if (listen_flag) {
      syslog(LOG_INFO, "starting in listening mode with serviceid=%s", serviceid);
      SOCKET lsocket = create_listening_socket(sid);
      sock = accept_socket(lsocket);
    } else {
      syslog(LOG_INFO, "starting in connect mode with serviceid=%s", serviceid);
      sock = connect_socket(sid);
    }

    handle(sock);
    exit(0);
}
