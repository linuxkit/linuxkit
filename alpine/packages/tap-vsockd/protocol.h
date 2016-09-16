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

/*
 * This should be bumped whenever we add something (like a feature or a
 * bugfix) and we wish the UI to be able to detect when to trigger a
 * reinstall.
 */
#define CURRENT_VERSION 13

extern struct init_message *create_init_message(void);
extern int read_init_message(int fd, struct init_message *ci);
extern int write_init_message(int fd, struct init_message *ci);
extern char *print_init_message(struct init_message *m);

/* Client -> Server command */
enum command {
	ethernet = 1,
};

extern int write_command(int fd, enum command *c);

/* Client -> Server command arguments */
struct ethernet_args {
	char uuid_string[36];
};

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

#endif /* _VMNET_PROTOCOL_H_ */
