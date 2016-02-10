#include <sys/types.h>
#include <errno.h>

#include <stdlib.h>
#include <string.h>

#include <stdio.h>
#include <unistd.h>
#include <fcntl.h>
#include <sys/socket.h>
#include <sys/wait.h>
#include <pthread.h>
#include <signal.h>

#define COPY_BUFSZ 65536

int debug = 0;

int save_trace;

// execvpe is in unistd.h on Linux.
int execvpe(const char *path, char *const argv[], char *const envp[]);

typedef struct {
  char * socket9p_root;
} parameters;

typedef struct {
  parameters * params;
  long id;
} connection_state;

typedef struct {
  char * descr;
  long connection;
  char * tag;
  int from;
  int to;
} copy_thread_state;

char * fusermount = "fusermount";

void * must_malloc(char *const descr, size_t size) {
  void * ptr;

  ptr = malloc(size);
  if (size != 0 && ptr == NULL) {
    perror(descr);
    exit(1);
  }

  return ptr;
}

char ** read_opts(connection_state * connection, char * buf) {
  int read_fd;
  char * read_path;
  int read_count;
  int optc = 1;
  char ** optv;

  if (asprintf(&read_path, "%s/connections/%ld/read",
               connection->params->socket9p_root, connection->id) == -1) {
    fprintf(stderr, "Couldn't allocate read path\n");
    exit(1);
  }

  read_fd = open(read_path, O_RDONLY);
  if (read_fd == -1) {
    fprintf(stderr, "For connection %ld, ", connection->id);
    perror("couldn't open read path");
    exit(1);
  }

  read_count = read(read_fd, buf, COPY_BUFSZ - 1);
  if (read_count == -1) {
    perror("read_opts error reading");
    exit(1);
  }
  buf[read_count] = 0x0;

  for (int i = 0; i < read_count; i++) {
    if (buf[i] == 0x0) optc++;
  }

  optv = must_malloc("read_opts optv", (optc + 1) * sizeof(void *));
  optv[0] = buf;
  optv[optc] = 0x0;

  int j = 1;
  for (int i = 0; i < read_count && j < optc; i++) {
    if (buf[i] == 0x0) {
      optv[j] = buf + i + 1;
      j++;
    }
  }

  free(read_path);

  return optv;
}

uint64_t message_id(char * message) {
  return *((uint64_t *) (message + 8));
}

int copy(copy_thread_state * copy_state) {
  int from = copy_state->from;
  int to = copy_state->to;
  char * descr = copy_state->descr;
  int read_count, write_count;
  long connection = copy_state->connection;
  char * tag = copy_state->tag;
  char * buf;

  buf = must_malloc(descr, COPY_BUFSZ);

  while(1) {
    read_count = read(from, buf, COPY_BUFSZ);
    if (read_count == -1) {
      fprintf(stderr, "copy %s: error reading ", descr);
      perror("");
      exit(1);
    }

    if (save_trace) {
      int trace_fd;
      char * trace_path;

      // TODO: check for errors
      asprintf(&trace_path, "/tmp/transfused.%ld.%s.%llu",
               connection, tag, message_id(buf));
      trace_fd = open(trace_path, O_WRONLY | O_CREAT, 0600);
      write(trace_fd, buf, read_count);
      close(trace_fd);
      free(trace_path);
    }

    write_count = write(to, buf, read_count);
    if (write_count == -1) {
      fprintf(stderr, "copy %s: error writing ", descr);
      perror("");
      exit(1);
    }

    if (write_count != read_count) {
      fprintf(stderr, "copy %s: read %d but only wrote %d\n",
              descr, read_count, write_count);
      exit(1);
    }
  }

  free(buf);
  return 0;
}

int copy_clean_from(copy_thread_state * copy_state) {
  int ret = copy(copy_state);

  if (close(copy_state->from)) {
    fprintf(stderr, "For %s, ", copy_state->descr);
    perror("couldn't close read fd");
    exit(1);
  }

  free(copy_state->descr);
  free(copy_state);

  return ret;
}

int copy_clean_to(copy_thread_state * copy_state) {
  int ret = copy(copy_state);

  if (close(copy_state->to)) {
    fprintf(stderr, "For %s, ", copy_state->descr);
    perror("couldn't close write fd");
    exit(1);
  }

  free(copy_state->descr);
  free(copy_state);

  return ret;
}

int recv_fd(int sock) {
  int ret;
  int fd = -1;
  char iochar;
  char buf[CMSG_SPACE(sizeof(fd))];

  struct msghdr msg;
  struct iovec vec;
  struct cmsghdr *cmsg;

  msg.msg_name=NULL;
  msg.msg_namelen=0;
  vec.iov_base=&iochar;
  vec.iov_len=1;
  msg.msg_iov=&vec;

  msg.msg_iovlen=1;

  msg.msg_control = buf;
  msg.msg_controllen = sizeof(buf);

  ret = recvmsg(sock, &msg, 0);

  if (ret == -1) {
    perror("recvmsg");
    exit(1);
  }

  if (ret > 0 && msg.msg_controllen > 0) {
    cmsg = CMSG_FIRSTHDR(&msg);
    if (cmsg->cmsg_level == SOL_SOCKET && (cmsg->cmsg_type == SCM_RIGHTS)) {
      fd = *(int*)CMSG_DATA(cmsg);
    } else {
      fprintf(stderr, "recvmsg: failed to receive an fd\n");
    }
  }

  return fd;
}

// optv must be null-terminated
int get_fuse_sock(char *const optv[]) {
  int optc;
  char ** argv;
  char * envp[2];
  pid_t fusermount_pid;
  int fuse_socks[2];
  int status;
  int fd;

  // prepare argv from optv
  for (optc = 0; optv[optc] != NULL; optc++) {}

  argv = must_malloc("fusermount argv",(optc + 2) * sizeof(char *));

  argv[0] = fusermount;
  memcpy(&argv[1], optv, (optc + 1) * sizeof(char *));

  // make the socket over which we'll be sent the FUSE socket fd
  if (socketpair(PF_UNIX, SOCK_STREAM, 0, fuse_socks)) {
    perror("Couldn't create FUSE socketpair");
    exit(1);
  }

  // prepare to exec the suid binary fusermount
  if (asprintf(&envp[0], "_FUSE_COMMFD=%d", fuse_socks[0]) == -1) {
    fprintf(stderr, "Couldn't allocate fusermount envp\n");
    exit(1);
  }
  envp[1] = 0x0;

  // fork and exec fusermount
  fusermount_pid = fork();
  if (!fusermount_pid) { // child
    if (execvpe(fusermount, argv, envp)) {
      perror("Failed to execute fusermount");
      exit(1);
    }
  }

  // parent
  free(argv);
  free(envp[0]);

  // close the end of the socket that we gave away
  if (close(fuse_socks[0])) {
    perror("Couldn't close unneeded fusermount socket");
    exit(1);
  }
  
  // wait for fusermount to return
  waitpid(fusermount_pid, &status, 0);
  if (!WIFEXITED(status)) {
    fprintf(stderr, "fusermount terminated abnormally\n");
    exit(1);
  }
  if (WEXITSTATUS(status)) {
    fprintf(stderr, "fusermount exited with code %d\n", WEXITSTATUS(status));
    exit(1);
  }

  if (debug) fprintf(stderr, "about to recv_fd from fusermount\n");

  fd = recv_fd(fuse_socks[1]);
  if (fd == -1) {
    fprintf(stderr, "Couldn't receive fd over FUSE socket\n");
    exit(1);
  }

  // close the read end of the socket
  if (close(fuse_socks[1])) {
    perror("Couldn't close fusermount read socket");
    exit(1);
  }

  return fd;
}

int start_reader(connection_state * connection, int fuse) {
  int read_fd;
  char * read_path;
  pthread_t child;
  copy_thread_state * copy_state;
  void *(*copy_clean)(void *) = (void *(*)(void *)) copy_clean_from;

  if (asprintf(&read_path, "%s/connections/%ld/read",
               connection->params->socket9p_root, connection->id) == -1) {
    fprintf(stderr, "Couldn't allocate read path\n");
    exit(1);
  }

  read_fd = open(read_path, O_RDONLY);
  if (read_fd == -1) {
    fprintf(stderr, "For connection %ld, ", connection->id);
    perror("couldn't open read path");
    exit(1);
  }

  copy_state = must_malloc("start_reader copy_state",
                           sizeof(copy_thread_state));
  copy_state->descr = read_path;
  copy_state->connection = connection->id;
  copy_state->tag = "read";
  copy_state->from = read_fd;
  copy_state->to = fuse;
  if ((errno = pthread_create(&child, NULL, copy_clean, copy_state))) {
    fprintf(stderr, "couldn't create read copy thread for connection %ld ",
            connection->id);
    perror("");
    exit(1);
  }

  if ((errno = pthread_detach(child))) {
    fprintf(stderr, "couldn't detach read copy thread for connection '%ld' ",
            connection->id);
    perror("");
    exit(1);
  }

  return 0;
}

int start_writer(connection_state * connection, int fuse) {
  int write_fd;
  char * write_path;
  copy_thread_state * copy_state;

  if (asprintf(&write_path, "%s/connections/%ld/write",
               connection->params->socket9p_root, connection->id) == -1) {
    fprintf(stderr, "Couldn't allocate write path\n");
    exit(1);
  }

  write_fd = open(write_path, O_WRONLY);
  if (write_fd == -1) {
    fprintf(stderr, "For connection %ld, ", connection->id);
    perror("couldn't open write path");
    exit(1);
  }

  copy_state = must_malloc("start_writer copy_state",
                           sizeof(copy_thread_state));
  copy_state->descr = write_path;
  copy_state->connection = connection->id;
  copy_state->tag = "write";
  copy_state->from = fuse;
  copy_state->to = write_fd;
  copy_clean_to(copy_state);

  return 0;
}

int handle_connection(connection_state * connection) {
  char ** optv;
  int fuse;
  char * buf;
  int ret;
  
  buf = must_malloc("read_opts packet malloc", COPY_BUFSZ);

  optv = read_opts(connection, buf);
  fuse = get_fuse_sock(optv);
  free(optv);
  free(buf);

  start_reader(connection, fuse);
  ret = start_writer(connection, fuse);
  free(connection);

  return ret;
}

void toggle_save_trace(int sig) {
  save_trace = !save_trace;
}

void setup_save_trace() {
  save_trace = 0;
  
  if (SIG_ERR == signal(SIGHUP, toggle_save_trace)) {
    perror("Couldn't set SIGHUP behavior");
    exit(1);
  }

  if (siginterrupt(SIGHUP, 1)) {
    perror("Couldn't set siginterrupt for SIGHUP");
    exit(1);
  }
}

#define ID_LEN 512

int main(int argc, char * argv[]) {
  int events, read_count;
  char buf[ID_LEN];
  long conn;
  pthread_t child;
  void *(*handle)(void *) = (void *(*)(void *)) handle_connection;
  parameters params;
  connection_state * connection;
  char * events_path;

  if (argc < 2) {
    params.socket9p_root = "/Transfuse";
  } else {
    params.socket9p_root = argv[1];
  }

  if (asprintf(&events_path, "%s/events", params.socket9p_root) == -1) {
    fprintf(stderr, "Couldn't allocate events path\n");
    exit(1);
  }

  setup_save_trace();

  events = open(events_path, O_RDONLY | O_CLOEXEC);
  while (events != -1) {
    read_count = read(events, buf, ID_LEN - 1);
    if (read_count == -1) {
      perror("Error reading events path");
      exit(1);
    } else if (read_count == 0) {
      // TODO: this is probably the 9p server's fault due to
      //       not dropping the read 0 to force short read if
      //       the real read is flushed
      fprintf(stderr, "read 0 from event stream\n");
      continue;
    }

    buf[read_count] = 0x0;

    errno = 0;
    conn = strtol(buf, NULL, 10);
    if (errno) {
      fprintf(stderr, "connection id of string '%s' ", buf);
      perror("failed");
      exit(1);
    }

    if (debug) fprintf(stderr, "handle connection %ld\n", conn);

    connection = must_malloc("connection state", sizeof(connection_state));
    connection->id = conn;
    connection->params = &params;

    if ((errno = pthread_create(&child, NULL, handle, connection))) {
      fprintf(stderr, "couldn't create thread for connection '%ld' ", conn);
      perror("");
      exit(1);
    }

    if ((errno = pthread_detach(child))) {
      fprintf(stderr, "couldn't detach thread for connection '%ld' ", conn);
      perror("");
      exit(1);
    }

    if (debug) fprintf(stderr, "thread spawned\n");
  }

  fprintf(stderr, "failed to open events path: %s\n", events_path);

  free(events_path);
  return 1;
}
