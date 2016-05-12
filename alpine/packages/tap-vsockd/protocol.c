#include <sys/socket.h>
#include <sys/types.h>
#include <sys/uio.h>
#include <unistd.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

#include "protocol.h"
#include "commit.h"

#include "utils.h"

/* Version 0 of the protocol used this */
char expected_hello_old[5] = { 'V', 'M', 'N', 'E', 'T' };

/* Version 1 and later of the protocol used this */
char expected_hello[5] = { 'V', 'M', 'N', '3', 'T' };

int really_read(int fd, uint8_t *buffer, size_t total){
  size_t remaining = total;
	ssize_t n;
  while (remaining > 0){
    n = read(fd, buffer, remaining);
    if (n == 0){
      aslLog(ASL_LEVEL_NOTICE, "EOF reading packet from Unix domain socket: closing");
      goto err;
    }
    if (n < 0){
      aslLog(ASL_LEVEL_NOTICE, "failure reading packet from Unix domain socket, closing: %s", strerror (errno));
      goto err;
    }
    remaining -= (size_t)n;
    buffer = buffer + n;
  }
  return 0;
err:
  /* On error: stop reading from the socket and trigger a clean shutdown */
  shutdown(fd, SHUT_RD);
  return -1;
}

int really_write(int fd, uint8_t *buffer, size_t total){
	size_t remaining = total;
  ssize_t n;
  while (remaining > 0){
    n = write(fd, buffer, remaining);
    if (n == 0){
      aslLog(ASL_LEVEL_INFO, "EOF writing to Unix domain socket");
      goto err;
    }
    if (n < 0){
      aslLog(ASL_LEVEL_NOTICE, "failure writing packet from Unix domain socket, closing: %s", strerror (errno));
      goto err;
    }
    remaining -= (size_t) n;
    buffer = buffer + n;
  }
  return 0;
err:
  /* On error: stop listening to the socket */
  shutdown(fd, SHUT_WR);
  return -1;
}

int really_writev(int fd, struct iovec *iov, int iovcnt){
	ssize_t n = 0;
	int pos = 0;

	while (pos < iovcnt) {
		iov[pos].iov_base = (char *)iov[pos].iov_base + n;
		iov[pos].iov_len -= (size_t)n;

		n = writev(fd, iov + pos, iovcnt - pos);

		if (n == 0){
			aslLog(ASL_LEVEL_INFO, "EOF writing to Unix domain socket");
			goto err;
		}
		if (n < 0){
			aslLog(ASL_LEVEL_NOTICE, "failure writing packet from Unix domain socket, closing: %s", strerror (errno));
			goto err;
		}

		// advance pos, reduce n
		for (; pos < iovcnt && (size_t)n >= iov[pos].iov_len; pos++) {
			n -= iov[pos].iov_len;
		}
	}
  return 0;
err:
  shutdown(fd, SHUT_WR);
  return -1;
}

struct init_message *create_init_message(){
  struct init_message *m = (struct init_message*) malloc(sizeof(struct init_message));
  bzero(m, sizeof(struct init_message));
  memcpy(&m->hello[0], &expected_hello[0], sizeof(m->hello));
  m->version = CURRENT_VERSION;
  memcpy(&m->commit[0], &commit[0], sizeof(m->commit));
  return m;
}

char *print_init_message(struct init_message *m) {
  char tmp[41];
  memcpy(&tmp[0], &m->commit[0], 40);
  tmp[40] = '\000';
  char *buffer = (char*) malloc(80);
  int n = snprintf(buffer, 80, "version %d, commit %s", m->version, tmp);
  if (n < 0) {
    aslLog(ASL_LEVEL_ERR, "Failed to format init_message: %s", strerror (errno));
    exit(1);
  }
  return buffer;
}

int read_init_message(int fd, struct init_message *ci) {
  bzero(ci, sizeof(struct init_message));
  if (really_read(fd, (uint8_t*) &ci->hello[0], sizeof(ci->hello)) == -1){
    aslLog(ASL_LEVEL_CRIT, "Failed to read hello from client");
    return -1;
  }
  if (memcmp(&ci->hello[0], &expected_hello_old[0], sizeof(expected_hello_old)) == 0) {
      ci->version = 0;
      return 0;
  }
  if (memcmp(&ci->hello[0], &expected_hello[0], sizeof(expected_hello)) != 0) {
    aslLog(ASL_LEVEL_ERR, "Failed to read header magic from client");
    return -1;
  }
  if (really_read(fd, (uint8_t*) &ci->version, sizeof(ci->version)) == -1){
    aslLog(ASL_LEVEL_ERR, "Failed to read header version from client");
    return -1;
  }
  if (really_read(fd, (uint8_t*) &ci->commit[0], sizeof(ci->commit)) == -1){
    aslLog(ASL_LEVEL_ERR, "Failed to read header hash from client");
    return -1;
  }
  return 0;
}

int write_init_message(int fd, struct init_message *ci) {
  if (really_write(fd, (uint8_t*) &ci->hello[0], sizeof(ci->hello)) == -1){
    aslLog(ASL_LEVEL_ERR, "Failed to write hello to client");
    return -1;
  }
  if (ci->version > 0) {
    if (really_write(fd, (uint8_t*) &ci->version, sizeof(ci->version)) == -1){
      aslLog(ASL_LEVEL_ERR, "Failed to write version to client");
      return -1;
    }
    if (really_write(fd, (uint8_t*) &ci->commit[0], sizeof(ci->commit)) == -1){
      aslLog(ASL_LEVEL_ERR, "Failed to write hash to client");
      return -1;
    }
  }
  return 0;
}

int read_vif_info(int fd, struct vif_info *vif) {
  uint8_t buffer[10];
  if (really_read(fd, &buffer[0], sizeof(buffer)) == -1){
    aslLog(ASL_LEVEL_ERR, "Failed to read vif info from client");
    return -1;
  }
  vif->mtu = (size_t) (buffer[0] | (buffer[1] << 8));
  vif->max_packet_size = (size_t) (buffer[2] | (buffer[3] << 8));
  memcpy(vif->mac, &buffer[4], 6);
  return 0;
}


int write_vif_info(int fd, struct vif_info *vif) {
  uint8_t buffer[10];
  buffer[0] = (uint8_t) ((vif->mtu >> 0) & 0xff);
  buffer[1] = (uint8_t) ((vif->mtu >> 8) & 0xff);
  buffer[2] = (uint8_t) ((vif->max_packet_size >> 0) & 0xff);
  buffer[3] = (uint8_t) ((vif->max_packet_size >> 8) & 0xff);
  memcpy(&buffer[0] + 4, &(vif->mac)[0], 6);
  if (really_write(fd, &buffer[0], sizeof(buffer)) == -1){
    aslLog(ASL_LEVEL_ERR, "Failed to write vif info to client");
    return -1;
  }
  return 0;
}

int read_command(int fd, enum command *c) {
  uint8_t command;
  if (really_read(fd, (uint8_t*) &command, sizeof(command)) == -1){
    aslLog(ASL_LEVEL_ERR, "Failed to read command from client");
    return -1;
  }
  if ((command != ethernet) && (command != uninstall) && (command != install_symlinks) && (command != uninstall_symlinks) && (command != uninstall_sockets) && (command != bind_ipv4)) {
    aslLog(ASL_LEVEL_CRIT, "Client sent unknown command: %d", command);
    return -1;
  }
  *c = command;
  return 0;
}

int write_command(int fd, enum command *c) {
  uint8_t command = *c;
  if (really_write(fd, (uint8_t*) &command, sizeof(command)) == -1){
    aslLog(ASL_LEVEL_ERR, "Failed to write command to client");
    return -1;
  }
  return 0;
}

int read_ethernet_args(int fd, struct ethernet_args *args){
  if (really_read(fd, (uint8_t*) &args->uuid_string[0], 36) == -1){
    aslLog(ASL_LEVEL_CRIT, "Failed to read ethernet args from client");
    return -1;
  }
  return 0;
}

int write_ethernet_args(int fd, struct ethernet_args *args){
  if (really_write(fd, (uint8_t*) &args->uuid_string[0], 36) == -1){
    aslLog(ASL_LEVEL_ERR, "Failed to write ethernet args to client");
    return -1;
  }
  return 0;
}

int read_bind_ipv4(int fd, struct bind_ipv4 *ip) {
  if (really_read(fd, (uint8_t*) &ip->ipv4, 4) == -1){
    aslLog(ASL_LEVEL_CRIT, "Failed to read IPv4 address from client");
    return -1;
  }
  if (really_read(fd, (uint8_t*) &ip->port, 2) == -1){
    aslLog(ASL_LEVEL_CRIT, "Failed to read port from client");
    return -1;
  }
  if (really_read(fd, (uint8_t*) &ip->stream, 1) == -1){
    aslLog(ASL_LEVEL_CRIT, "Failed to read stream from client");
    return -1;
  }
  if ((ip->stream != 0) && (ip->stream != 1)) {
    aslLog(ASL_LEVEL_CRIT, "Failed to parse stream value: %d", ip->stream);
    return -1;
  }

  return 0;
}

int write_bind_ipv4(int fd, struct bind_ipv4 *ip) {
  if (really_write(fd, (uint8_t*) &ip->ipv4, 4) == -1){
    aslLog(ASL_LEVEL_ERR, "Failed to write IPv4 to server");
    return -1;
  }
  if (really_write(fd, (uint8_t*) &ip->port, 2) == -1){
    aslLog(ASL_LEVEL_ERR, "Failed to write port to server");
    return -1;
  }
  if (really_write(fd, (uint8_t*) &ip->stream, 1) == -1){
    aslLog(ASL_LEVEL_ERR, "Failed to write stream to server");
    return -1;
  }
  return 0;
}
