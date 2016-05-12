#ifndef _VMNET_PROTOCOL_H_
#define _VMNET_PROTOCOL_H_

#include <errno.h>
#include <stdint.h>

/* Client -> Server init_message */
/* Server -> Client init_message */
struct init_message {
  char hello[5];
  uint8_t _padding[3];
  uint32_t version;
  char commit[40]; /* git sha of the compiled commit */
};

/* This should be bumped whenever we add something (like a feature or a bugfix)
   and we wish the UI to be able to detect when to trigger a reinstall. */
#define CURRENT_VERSION 13

extern struct init_message *create_init_message(void);
extern int read_init_message(int fd, struct init_message *ci);
extern int write_init_message(int fd, struct init_message *ci);
extern char *print_init_message(struct init_message *m);

/* Client -> Server command */
enum command {
  ethernet = 1,
  uninstall = 2,
  install_symlinks = 3,
  uninstall_symlinks = 4,
  uninstall_sockets = 5, // to uninstall all sockets but com.docker.vmnetd.socket
  bind_ipv4 = 6,
};

extern int read_command(int fd, enum command *c);
extern int write_command(int fd, enum command *c);

/* Client -> Server command arguments */
struct ethernet_args {
  char uuid_string[36];
};

extern int read_ethernet_args(int fd, struct ethernet_args *args);
extern int write_ethernet_args(int fd, struct ethernet_args *args);

/* Server -> Client: details of a vif */
struct vif_info {
  uint8_t mac[6];
  short _padding;
  size_t max_packet_size;
  size_t mtu;
};

extern int read_vif_info(int fd, struct vif_info *vif);
extern int write_vif_info(int fd, struct vif_info *vif);

extern char expected_hello[5];
extern char expected_hello_old[5];

extern int really_read(int fd, uint8_t *buffer, size_t total);
extern int really_write(int fd, uint8_t *buffer, size_t total);
extern int really_writev(int fd, struct iovec *iov, int iovcnt);

// Client -> Server: requested IPv4 address and port
struct bind_ipv4 {
  uint32_t ipv4;
  short _padding;
  uint16_t port;
  uint8_t stream; /* 0 = stream; 1 = dgram */
  uint8_t _padding2[3];
};

extern int read_bind_ipv4(int fd, struct bind_ipv4 *vif);
extern int write_bind_ipv4(int fd, struct bind_ipv4 *vif);

#endif /* _VMNET_PROTOCOL_H_ */
