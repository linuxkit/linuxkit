#include <stddef.h>
#include <stdlib.h>

#include <sys/socket.h>

#include "include/uapi/linux/vm_sockets.h"

#include "transfused_log.h"

long parse_cid(const char * address) {
  char * end = NULL;
  long cid = strtol(address, &end, 10);
  if (address == end || *end != ':') {
    *end = 0;
    die(2, NULL, NULL, "Invalid vsock cid: %s", address);
  }
  return cid;
}

long parse_port(const char * port_str) {
  char * end = NULL;
  long port = strtol(port_str, &end, 10);
  if (port_str == end || *end != '\0') {
    *end = 0;
    die(2, NULL, NULL, "Invalid vsock port: %s", port_str);
  }
  return port;
}

int find_colon(const char * address) {
  int colon = 0;

  while (address[colon] != '\0')
    if (address[colon] == ':') break;
    else colon++;

  if (address[colon] == '\0')
    die(2, NULL, NULL, "Missing port in vsock address %s", address);

  return colon;
}

int bind_vsock(const char * address) {
  long cid, port;
  int colon;

  struct sockaddr_vm sa_listen = {
    .svm_family = AF_VSOCK,
  };

  int sock_fd;

  colon = find_colon(address);

  if (address[0] == '_' && colon == 1) cid = VMADDR_CID_ANY;
  else cid = parse_cid(address);

  port = parse_port(address + colon + 1);

  sa_listen.svm_cid = cid;
  sa_listen.svm_port = port;

  sock_fd = socket(AF_VSOCK, SOCK_STREAM, 0);
  if (sock_fd < 0)
    die(1, NULL, "socket(AF_VSOCK)", "");

  if (bind(sock_fd, (struct sockaddr *) &sa_listen, sizeof(sa_listen)))
    die(1, NULL, "bind(AF_VSOCK)", "");

  return sock_fd;
}

int connect_vsock(const char * address) {
  long cid, port;
  int colon;

  struct sockaddr_vm sa_connect = {
    .svm_family = AF_VSOCK,
  };

  int sock_fd;

  colon = find_colon(address);

  cid = parse_cid(address);
  port = parse_port(address + colon + 1);

  sa_connect.svm_cid = cid;
  sa_connect.svm_port = port;

  sock_fd = socket(AF_VSOCK, SOCK_STREAM, 0);
  if (sock_fd < 0)
    die(1, NULL, "socket(AF_VSOCK)", "");

  if (connect(sock_fd, (struct sockaddr *) &sa_connect, sizeof(sa_connect)))
    die(1, NULL, "connect(AF_VSOCK)", "");

  return sock_fd;
}
