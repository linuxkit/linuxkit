#include <pthread.h>
#include <sys/socket.h>

typedef struct {
  char * server;
  char * socket;
  char * fusermount;
  char * pidfile;
  char * logfile;
  char * mount_trigger;
  char * trigger_log;
  int sock;
  pthread_mutex_t fd_lock;
  int logfile_fd;
  int trigger_fd;
} parameters;

typedef struct {
  parameters * params;
  char * type_descr;
  char * mount_point;
  struct sockaddr sa_client;
  socklen_t socklen_client;
  int sock;
} connection_t;

pthread_attr_t detached;

void * must_malloc(char *const descr, size_t size);

void lock(char *const descr, pthread_mutex_t * mutex);

void unlock(char *const descr, pthread_mutex_t * mutex);
