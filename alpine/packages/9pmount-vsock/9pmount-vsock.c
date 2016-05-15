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

#include "hvsock.h"

#define NONE 0
#define LISTEN 1
#define CONNECT 2

int mode = NONE;

char *default_sid = "C378280D-DA14-42C8-A24E-0DE92A1028E2";
char *mount = "/bin/mount";

void fatal(const char *msg)
{
    syslog(LOG_CRIT, "%s Error: %d. %s", msg, errno, strerror(errno));
    exit(1);
}

static void handle(int fd)
{
  char *options = NULL;
  if (asprintf(&options, "trans=fd,dfltuid=1001,dfltgid=50,version=9p2000,rfdno=%d,wfdno=%d", fd, fd) < 0){
    fatal("asprintf()");
  }
  char *argv[] = {
    mount,
    "-t", "9p", "-o", options,
    "db", "/Database",
    NULL
  };
  execv(mount, argv);
  fatal("execv()");
}

static int create_listening_socket(GUID serviceid) {
  int lsock = -1;
  SOCKADDR_HV sa;
  int res;

  lsock = socket(AF_HYPERV, SOCK_STREAM, HV_PROTOCOL_RAW);
  if (lsock == -1) {
    fatal("socket()");
  }

  sa.Family = AF_HYPERV;
  sa.Reserved = 0;
  sa.VmId = HV_GUID_WILDCARD;
  sa.ServiceId = serviceid;

  res = bind(lsock, (const struct sockaddr *)&sa, sizeof(sa));
  if (res == -1) {
    fatal("bind()");
  }

  res = listen(lsock, 1);
  if (res == -1) {
    fatal("listen()");
  }
  return lsock;
}

static int connect_socket(GUID serviceid) {
  int sock = -1;
  SOCKADDR_HV sa;
  int res;

  sock = socket(AF_HYPERV, SOCK_STREAM, HV_PROTOCOL_RAW);
  if (sock == -1) {
    fatal("socket()");
  }

  sa.Family = AF_HYPERV;
  sa.Reserved = 0;
  sa.VmId = HV_GUID_PARENT;
  sa.ServiceId = serviceid;

  res = connect(sock, (const struct sockaddr *)&sa, sizeof(sa));
  if (res == -1) {
    fatal("connect()");
  }

  return sock;
}

static int accept_socket(int lsock) {
  int csock = -1;
  SOCKADDR_HV sac;
  socklen_t socklen = sizeof(sac);

  csock = accept(lsock, (struct sockaddr *)&sac, &socklen);
  if (csock == -1) {
    fatal("accept()");
  }

  syslog(LOG_INFO, "Connect from: "GUID_FMT":"GUID_FMT"\n",
    GUID_ARGS(sac.VmId), GUID_ARGS(sac.ServiceId));
  return csock;
}

void usage(char *name)
{
    printf("%s usage:\n", name);
    printf("\t[--serviceid <guid>] <listen | connect>\n");
    printf("where\n");
    printf("\t--serviceid <guid>: use <guid> as the well-known service GUID\n");
    printf("\t  (defaults to %s)\n", default_sid);
    printf("\t--listen: listen forever for incoming AF_HVSOCK connections\n");
    printf("\t--connect: connect to the parent partition\n");
}

int main(int argc, char **argv)
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
    if (optind < argc) {
      if (strcmp(argv[optind], "listen") == 0) {
        mode = LISTEN;
      } else if (strcmp(argv[optind], "connect") == 0) {
        mode = CONNECT;
      }
      optind++;
    }
    if (mode == NONE) {
      fprintf(stderr, "Please supply either listen or connect\n");
      usage(argv[0]);
      exit(1);
    }
    openlog(argv[0], LOG_CONS | LOG_NDELAY | LOG_PERROR, LOG_DAEMON);

    res = parseguid(serviceid, &sid);
    if (res) {
      syslog(LOG_CRIT, "Failed to parse serviceid as GUID: %s", serviceid);
      usage(argv[0]);
      exit(1);
    }

    int sock = -1;
    if (mode == LISTEN) {
      syslog(LOG_INFO, "starting in listening mode with serviceid=%s", serviceid);
      int lsocket = create_listening_socket(sid);
      sock = accept_socket(lsocket);
      close(lsocket);
    } else {
      syslog(LOG_INFO, "starting in connect mode with serviceid=%s", serviceid);
      sock = connect_socket(sid);
    }

    handle(sock);
    exit(0);
}
