#ifndef _TRANSFUSED_H_
#define _TRANSFUSED_H_

#include <pthread.h>
#include <sys/socket.h>
#include "transfused_perfstat.h"

#define IN_BUFSZ  ((1 << 20) + 16)
#define OUT_BUFSZ ((1 << 20) + 64)
#define EVENT_BUFSZ 4096
#define CTL_BUFSZ 65536
#define PERFSTATS_PER_SEGMENT 2730 /* (64k - 16) / 24 */
#define MAX_PERFSTAT_CHECK 64

#define DEFAULT_FUSERMOUNT "/bin/fusermount"
#define DEFAULT_SOCKET "v:_:1525"
#define DEFAULT_SERVER "v:2:1524"

#define PING             128
#define RMDIR_SYSCALL    0
#define UNLINK_SYSCALL   1
#define MKDIR_SYSCALL    2
#define SYMLINK_SYSCALL  3
#define TRUNCATE_SYSCALL 4
#define CHMOD_SYSCALL    5
#define MKNOD_REG_SYSCALL 6
/* these could be turned into an enum probably but...C standard nausea */

#define MOUNT_SUITABILITY_REQUEST 1
#define EXPORT_SUITABILITY_REQUEST 2
#define START_PERFSTAT_REQUEST 3
#define STOP_PERFSTAT_REQUEST 4

#define TRANSFUSE_LOG_ERROR 1
#define TRANSFUSE_LOG_NOTICE 2
#define PONG_REPLY 3
#define MOUNT_SUITABILITY_REPLY 4
#define TRANSFUSE_NOTIFY_CHANNEL 5
#define PERFSTAT_REPLY 6
#define ERROR_REPLY 7

struct parameters;
struct connection;

struct parameters {
	char *server;
	char *socket;
	char *fusermount;
	char *pidfile;
	char *logfile;
	int logfile_fd;
	int ctl_sock;
	int data_sock;
	pthread_mutex_t ctl_lock;
	struct connection *connections;
};

typedef struct parameters parameters_t;

struct connection {
	struct connection *next;
	parameters_t *params;
	char *type_descr;
	char *mount_point;
	struct sockaddr sa_client;
	socklen_t socklen_client;
	int sock;
	int perfstat;
	perfstats_t *perfstats;
	pthread_mutex_t perfstat_lock;
};

typedef struct connection connection_t;

pthread_attr_t detached;

void *must_malloc(char *const descr, size_t size);
void lock(char *const descr, pthread_mutex_t *mutex);
void unlock(char *const descr, pthread_mutex_t *mutex);
void write_exactly(char *descr, int fd, void *buf, size_t nbyte);

void *error_reply(uint16_t id, const char *fmt, ...);

connection_t *find_connection(connection_t *conn, char *name, size_t len);

#endif /* _TRANSFUSED_H_ */
