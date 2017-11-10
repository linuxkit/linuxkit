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
#define CURRENT_VERSION 22

extern struct init_message *create_init_message(void);
extern int read_init_message(int fd, struct init_message *ci);
extern int write_init_message(int fd, struct init_message *ci);
extern char *print_init_message(struct init_message *m);

/* Client -> Server command */
enum command {
	ethernet = 1,
};

/* Server -> Client response */
enum response_type {
    rt_vif = 1,
    rt_disconnect = 2,
};

extern int write_command(int fd, enum command *c);

/* Client -> Server command arguments */
struct ethernet_args {
	char uuid_string[36];
};

extern int write_ethernet_args(int fd, struct ethernet_args *args);

/* Server -> Client: details of a vif */
struct vif_info {
	uint16_t mtu;
	uint16_t max_packet_size;
	uint8_t mac[6];
} __attribute__((packed));

/* Server -> Client: disconnect w/reason */
struct disconnect_reason {
    uint8_t len;
    char msg[256];
} __attribute__((packed));

struct msg_response {
    uint8_t response_type;
    union {
        struct vif_info vif;
        struct disconnect_reason disconnect;
    };
} __attribute__((packed));

extern int read_vif_response(int fd, struct vif_info *vif);

extern char expected_hello[5];
extern char expected_hello_old[5];

extern int really_read(int fd, uint8_t *buffer, size_t total);
extern int really_write(int fd, uint8_t *buffer, size_t total);

#endif /* _VMNET_PROTOCOL_H_ */
