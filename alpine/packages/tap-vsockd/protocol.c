#include <sys/socket.h>
#include <sys/types.h>
#include <sys/uio.h>
#include <unistd.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <syslog.h>

#include "protocol.h"

/* Version 0 of the protocol used this */
char expected_hello_old[5] = {'V', 'M', 'N', 'E', 'T'};

/* Version 1 and later of the protocol used this */
char expected_hello[5] = {'V', 'M', 'N', '3', 'T'};

int really_read(int fd, uint8_t *buffer, size_t total)
{
	size_t remaining = total;
	ssize_t n;

	while (remaining > 0) {
		n = read(fd, buffer, remaining);
		if (n == 0) {
			syslog(LOG_CRIT, "EOF reading from socket: closing\n");
			goto err;
		}
		if (n < 0) {
			syslog(LOG_CRIT,
			       "Failure reading from socket: closing: %s",
			       strerror(errno));
			goto err;
		}
		remaining -= (size_t) n;
		buffer = buffer + n;
	}
	return 0;
err:
	/*
	 * On error: stop reading from the socket and trigger a clean
	 * shutdown
	 */
	shutdown(fd, SHUT_RD);
	return -1;
}

int really_write(int fd, uint8_t *buffer, size_t total)
{
	size_t remaining = total;
	ssize_t n;

	while (remaining > 0) {
		n = write(fd, buffer, remaining);
		if (n == 0) {
			syslog(LOG_CRIT, "EOF writing to socket: closing");
			goto err;
		}
		if (n < 0) {
			syslog(LOG_CRIT,
			       "Failure writing to socket: closing: %s",
			       strerror(errno));
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

struct init_message *create_init_message()
{
	struct init_message *m;

	m = malloc(sizeof(struct init_message));
	if (!m)
		return NULL;

	bzero(m, sizeof(struct init_message));
	memcpy(&m->hello[0], &expected_hello[0], sizeof(m->hello));
	m->version = CURRENT_VERSION;
	memset(&m->commit[0], 0, sizeof(m->commit));

	return m;
}

char *print_init_message(struct init_message *m)
{
	char tmp[41];

	memcpy(&tmp[0], &m->commit[0], 40);
	tmp[40] = '\000';
	char *buffer;
	int n;

	buffer = malloc(80);
	if (!buffer)
		return NULL;

	n = snprintf(buffer, 80, "version %d, commit %s", m->version, tmp);
	if (n < 0) {
		perror("Failed to format init_message");
		exit(1);
	}
	return buffer;
}

int read_init_message(int fd, struct init_message *ci)
{
	int res;

	bzero(ci, sizeof(struct init_message));

	res = really_read(fd, (uint8_t *)&ci->hello[0], sizeof(ci->hello));
	if (res  == -1) {
		syslog(LOG_CRIT, "Failed to read hello from client");
		return -1;
	}

	res = memcmp(&ci->hello[0],
		     &expected_hello_old[0], sizeof(expected_hello_old));
	if (res == 0) {
		ci->version = 0;
		return 0;
	}

	res = memcmp(&ci->hello[0],
		     &expected_hello[0], sizeof(expected_hello));
	if (res != 0) {
		syslog(LOG_CRIT, "Failed to read header magic from client");
		return -1;
	}

	res = really_read(fd, (uint8_t *)&ci->version, sizeof(ci->version));
	if (res == -1) {
		syslog(LOG_CRIT, "Failed to read header version from client");
		return -1;
	}

	res = really_read(fd, (uint8_t *)&ci->commit[0], sizeof(ci->commit));
	if (res == -1) {
		syslog(LOG_CRIT, "Failed to read header hash from client");
		return -1;
	}

	return 0;
}

int write_init_message(int fd, struct init_message *ci)
{
	int res;

	res = really_write(fd, (uint8_t *)&ci->hello[0], sizeof(ci->hello));
	if (res == -1) {
		syslog(LOG_CRIT, "Failed to write hello to client");
		return -1;
	}
	if (ci->version > 0) {
		res = really_write(fd, (uint8_t *)&ci->version,
				   sizeof(ci->version));
		if (res == -1) {
			syslog(LOG_CRIT, "Failed to write version to client");
			return -1;
		}
		res = really_write(fd, (uint8_t *)&ci->commit[0],
				   sizeof(ci->commit));
		if (res == -1) {
			syslog(LOG_CRIT,
			       "Failed to write header hash to client");
			return -1;
		}
	}
	return 0;
}

int read_vif_info(int fd, struct vif_info *vif)
{
	uint8_t buffer[10];

	if (really_read(fd, &buffer[0], sizeof(buffer)) == -1) {
		syslog(LOG_CRIT, "Failed to read vif info from client");
		return -1;
	}

	vif->mtu = (size_t)(buffer[0] | (buffer[1] << 8));
	vif->max_packet_size = (size_t)(buffer[2] | (buffer[3] << 8));
	memcpy(vif->mac, &buffer[4], 6);
	return 0;
}


int write_vif_info(int fd, struct vif_info *vif)
{
	uint8_t buffer[10];

	buffer[0] = (uint8_t) ((vif->mtu >> 0) & 0xff);
	buffer[1] = (uint8_t) ((vif->mtu >> 8) & 0xff);
	buffer[2] = (uint8_t) ((vif->max_packet_size >> 0) & 0xff);
	buffer[3] = (uint8_t) ((vif->max_packet_size >> 8) & 0xff);
	memcpy(&buffer[0] + 4, &(vif->mac)[0], 6);

	if (really_write(fd, &buffer[0], sizeof(buffer)) == -1) {
		syslog(LOG_CRIT, "Failed to write vif into to client");
		return -1;
	}
	return 0;
}

int write_command(int fd, enum command *c)
{
	uint8_t command = *c;

	if (really_write(fd, (uint8_t *)&command, sizeof(command)) == -1) {
		syslog(LOG_CRIT, "Failed to write command to client");
		return -1;
	}
	return 0;
}

int write_ethernet_args(int fd, struct ethernet_args *args)
{
	if (really_write(fd, (uint8_t *)&args->uuid_string[0], 36) == -1) {
		syslog(LOG_CRIT, "Failed to write ethernet args to client");
		return -1;
	}
	return 0;
}
