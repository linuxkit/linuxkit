#include <sys/types.h>
#include <errno.h>

#include <stdlib.h>
#include <string.h>
#include <stdarg.h>
#include <syslog.h>

#include <stdio.h>
#include <unistd.h>
#include <fcntl.h>
#include <sys/socket.h>
#include <sys/wait.h>
#include <pthread.h>
#include <signal.h>

#include <netinet/in.h>
#include <inttypes.h>
#include <sys/stat.h>

#define COPY_BUFSZ 65536

#define DEFAULT_FUSERMOUNT "/bin/fusermount"
#define DEFAULT_SOCKET9P_ROOT "/Transfuse"

#define RMDIR_SYSCALL    0
#define UNLINK_SYSCALL   1
#define MKDIR_SYSCALL    2
#define SYMLINK_SYSCALL  3
#define TRUNCATE_SYSCALL 4
// these could be turned into an enum probably but... C standard nausea

char * default_fusermount = DEFAULT_FUSERMOUNT;
char * default_socket9p_root = DEFAULT_SOCKET9P_ROOT;
char * usage =
  "usage: transfused [-p pidfile] [-9 socket9p_root] [-f fusermount]\n"
  " -p pidfile\tthe path at which to write the pid of the process\n"
  " -9 " DEFAULT_SOCKET9P_ROOT "\tthe root of the 9p socket file system\n"
  " -f " DEFAULT_FUSERMOUNT "\tthe fusermount executable to use\n";

int debug = 0;

typedef struct {
  char * socket9p_root;
  char * fusermount;
  char * pidfile;
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

#include <sys/syscall.h>
#ifdef SYS_gettid
pid_t gettid() {
  return syscall(SYS_gettid);
}
#else
#error "SYS_gettid not defined"
#endif

void die(int exit_code, const char * perror_arg, const char * fmt, ...) {
  va_list argp;
  int in_errno = errno;
  va_start(argp, fmt);
  vsyslog(LOG_CRIT, fmt, argp);
  va_end(argp);
  if (perror_arg != NULL) {
    if (*perror_arg != 0)
      syslog(LOG_CRIT, "%s: %s", perror_arg, strerror(in_errno));
    else
      syslog(LOG_CRIT, "%s", strerror(in_errno));
  }
  exit(exit_code);
}

void * must_malloc(char *const descr, size_t size) {
  void * ptr;

  ptr = malloc(size);
  if (size != 0 && ptr == NULL) die(1, descr, "");

  return ptr;
}

char ** read_opts(connection_state * connection, char * buf) {
  int read_fd;
  char * read_path;
  int read_count;
  int optc = 1;
  char ** optv;

  if (asprintf(&read_path, "%s/connections/%ld/read",
               connection->params->socket9p_root, connection->id) == -1)
    die(1, "Couldn't allocate read path", "");

  read_fd = open(read_path, O_RDONLY);
  if (read_fd == -1)
    die(1, "couldn't open read path", "For connection %ld, ", connection->id);

  read_count = read(read_fd, buf, COPY_BUFSZ - 1);
  if (read_count == -1) die(1, "read_opts error reading", "");

  buf[read_count] = 0x0;

  for (int i = 0; i < read_count; i++) {
    if (buf[i] == 0x0) optc++;
  }

  optv = (char **) must_malloc("read_opts optv", (optc + 1) * sizeof(void *));
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

uint64_t message_id(uint64_t * message) {
  return message[1];
}

void copy(copy_thread_state * copy_state) {
  int from = copy_state->from;
  int to = copy_state->to;
  char * descr = copy_state->descr;
  int read_count, write_count;
  long connection = copy_state->connection;
  char * tag = copy_state->tag;
  void * buf;

  buf = must_malloc(descr, COPY_BUFSZ);

  while(1) {
    read_count = read(from, buf, COPY_BUFSZ);
    if (read_count == -1) die(1, "", "copy %s: error reading: ", descr);

    if (debug) {
      int trace_fd;
      char * trace_path;

      if (asprintf(&trace_path, "/tmp/transfused.%ld.%s.%llu",
                   connection, tag, message_id(buf)) == -1)
        die(1, "Couldn't allocate trace packet path", "");

      trace_fd = open(trace_path, O_WRONLY | O_CREAT, 0600);
      if (trace_fd == -1)
        die(1, "couldn't open trace packet path", "For %s, ", descr);

      write_count = write(trace_fd, buf, read_count);
      if (write_count == -1)
        die(1, "", "copy %s trace: error writing %s: ", descr, trace_path);

      if (write_count != read_count)
        die(1, NULL, "copy %s trace: read %d but only wrote %d\n",
            descr, read_count, write_count);

      close(trace_fd);
      free(trace_path);
    }

    write_count = write(to, buf, read_count);
    if (write_count == -1) die(1, "", "copy %s: error writing: ", descr);

    if (write_count != read_count)
      die(1, NULL, "copy %s: read %d but only wrote %d\n",
          descr, read_count, write_count);
  }

  free(buf);
}

void * copy_clean_from(copy_thread_state * copy_state) {
  copy(copy_state);

  close(copy_state->from);

  free(copy_state->descr);
  free(copy_state);

  return NULL;
}

void * copy_clean_from_thread(void * copy_state) {
  return (copy_clean_from((copy_thread_state *) copy_state));
}

void * copy_clean_to(copy_thread_state * copy_state) {
  copy(copy_state);

  close(copy_state->to);

  free(copy_state->descr);
  free(copy_state);

  return NULL;
}

int recv_fd(int sock) {
  int ret;
  int fd = -1;
  char iochar;
  char buf[CMSG_SPACE(sizeof(fd))];

  struct msghdr msg;
  struct iovec vec;
  struct cmsghdr *cmsg;

  msg.msg_name = NULL;
  msg.msg_namelen = 0;
  vec.iov_base = &iochar;
  vec.iov_len = 1;
  msg.msg_iov = &vec;

  msg.msg_iovlen = 1;

  msg.msg_control = buf;
  msg.msg_controllen = sizeof(buf);

  ret = recvmsg(sock, &msg, 0);

  if (ret == -1) die(1, "recvmsg", "");

  if (ret > 0 && msg.msg_controllen > 0) {
    cmsg = CMSG_FIRSTHDR(&msg);
    if (cmsg->cmsg_level == SOL_SOCKET && (cmsg->cmsg_type == SCM_RIGHTS)) {
      fd = *(int*)CMSG_DATA(cmsg);
    }
  }

  return fd;
}

// optv must be null-terminated
int get_fuse_sock(char * fusermount, char *const optv[]) {
  int optc;
  char ** argv;
  char * envp[2];
  pid_t fusermount_pid;
  int fuse_socks[2];
  int status;
  int fd;

  // prepare argv from optv
  for (optc = 0; optv[optc] != NULL; optc++) {}

  argv = (char **) must_malloc("fusermount argv", (optc + 2) * sizeof(char *));

  argv[0] = fusermount;
  memcpy(&argv[1], optv, (optc + 1) * sizeof(char *));

  // make the socket over which we'll be sent the FUSE socket fd
  if (socketpair(PF_UNIX, SOCK_STREAM, 0, fuse_socks))
    die(1, "Couldn't create FUSE socketpair", "");

  // prepare to exec the suid binary fusermount
  if (asprintf(&envp[0], "_FUSE_COMMFD=%d", fuse_socks[0]) == -1)
    die(1, "Couldn't allocate fusermount envp", "");

  envp[1] = 0x0;

  // fork and exec fusermount
  fusermount_pid = fork();
  if (!fusermount_pid) // child
    if (execve(fusermount, argv, envp))
      die(1, "Failed to execute fusermount", "");

  // parent
  free(argv);
  free(envp[0]);

  // close the end of the socket that we gave away
  close(fuse_socks[0]);

  // wait for fusermount to return
  waitpid(fusermount_pid, &status, 0);
  if (!WIFEXITED(status))
    die(1, NULL, "fusermount terminated abnormally\n");

  if (WEXITSTATUS(status))
    die(1, NULL, "fusermount exited with code %d\n", WEXITSTATUS(status));

  if (debug) syslog(LOG_DEBUG, "about to recv_fd from fusermount");

  fd = recv_fd(fuse_socks[1]);
  if (fd == -1)
    die(1, NULL, "Couldn't receive fd over FUSE socket\n");

  // close the read end of the socket
  close(fuse_socks[1]);

  return fd;
}

void start_reader(connection_state * connection, int fuse) {
  int read_fd;
  char * read_path;
  pthread_t child;
  copy_thread_state * copy_state;

  if (asprintf(&read_path, "%s/connections/%ld/read",
               connection->params->socket9p_root, connection->id) == -1)
    die(1, "Couldn't allocate read path", "");

  read_fd = open(read_path, O_RDONLY);
  if (read_fd == -1)
    die(1, "couldn't open read path", "For connection %ld, ", connection->id);

  copy_state = (copy_thread_state *) must_malloc("start_reader copy_state",
                                                 sizeof(copy_thread_state));
  copy_state->descr = read_path;
  copy_state->connection = connection->id;
  copy_state->tag = "read";
  copy_state->from = read_fd;
  copy_state->to = fuse;
  if ((errno = pthread_create(&child, NULL,
                              copy_clean_from_thread, copy_state)))
    die(1, "", "couldn't create read copy thread for connection %ld: ",
        connection->id);

  if ((errno = pthread_detach(child)))
    die (1, "", "couldn't detach read copy thread for connection '%ld': ",
         connection->id);
}

void do_write(connection_state * connection, int fuse) {
  int write_fd;
  char * write_path;
  copy_thread_state * copy_state;

  if (asprintf(&write_path, "%s/connections/%ld/write",
               connection->params->socket9p_root, connection->id) == -1)
    die(1, "Couldn't allocate write path", "");

  write_fd = open(write_path, O_WRONLY);
  if (write_fd == -1)
    die(1, "couldn't open write path", "For connection %ld, ", connection->id);

  copy_state = (copy_thread_state *) must_malloc("do_write copy_state",
                                                 sizeof(copy_thread_state));
  copy_state->descr = write_path;
  copy_state->connection = connection->id;
  copy_state->tag = "write";
  copy_state->from = fuse;
  copy_state->to = write_fd;
  copy_clean_to(copy_state);
}

void * mount_connection(connection_state * connection) {
  char ** optv;
  int fuse;
  char * buf;
  
  buf = (char *) must_malloc("read_opts packet malloc", COPY_BUFSZ);

  optv = read_opts(connection, buf);
  fuse = get_fuse_sock(connection->params->fusermount, optv);
  free(optv);
  free(buf);

  start_reader(connection, fuse);
  do_write(connection, fuse);
  free(connection);

  return NULL;
}

void * mount_thread(void * connection) {
  return mount_connection((connection_state *) connection);
}

void write_pid(connection_state * connection) {
  int write_fd;
  char * write_path;
  pid_t pid = gettid();
  char * pid_s;
  int pid_s_len, write_count;

  if (asprintf(&write_path, "%s/connections/%ld/write",
               connection->params->socket9p_root, connection->id) == -1)
    die(1, "Couldn't allocate write path", "");

  write_fd = open(write_path, O_WRONLY);
  if (write_fd == -1)
    die(1, "couldn't open write path", "For connection %ld, ", connection->id);

  if (asprintf(&pid_s, "%lld", (long long) pid) == -1)
    die(1, "Couldn't allocate pid string", "");

  pid_s_len = strlen(pid_s);

  write_count = write(write_fd, pid_s, pid_s_len);
  if (write_count == -1)
    die(1, "Error writing pid", "");

  if (write_count != pid_s_len)
    die(1, NULL, "Error writing pid %s to socket: only wrote %d bytes\n",
        pid_s, write_count);

  close(write_fd);
  free(pid_s);
  free(write_path);
}

void perform_syscall(uint8_t syscall, char path[]) {
  char * name;
  int r = 0;

  switch (syscall) {

  case RMDIR_SYSCALL:
    name = "rmdir";
    r = rmdir(path);
    break;

  case UNLINK_SYSCALL:
    name = "unlink";
    r = unlink(path);
    break;

  case MKDIR_SYSCALL:
    name = "mkdir";
    r = mkdir(path, 00000);
    break;

  case SYMLINK_SYSCALL:
    name = "symlink";
    r = symlink(".",path);
    break;

  case TRUNCATE_SYSCALL:
    name = "truncate";
    r = truncate(path, 0);
    break;

  default:
    die(1, NULL, "Unknown event syscall %" PRIu8, syscall);
  }

  if (r != 0) syslog(LOG_INFO, "Event %s error: %s", name, strerror(errno));
}

void * event_thread(void * connection_ptr) {
  char * read_path;
  int read_fd;
  int read_count, event_len, path_len;
  void * buf;
  connection_state * connection = connection_ptr;

  char * path;
  uint8_t syscall;
  void * msg;

  write_pid(connection);

  if (asprintf(&read_path, "%s/connections/%ld/read",
               connection->params->socket9p_root, connection->id) == -1)
    die(1, "Couldn't allocate read path", "");

  read_fd = open(read_path, O_RDONLY);
  if (read_fd == -1)
    die(1, "couldn't open read path", "For connection %ld, ", connection->id);

  buf = must_malloc("incoming event buffer", COPY_BUFSZ);

  while(1) {
    read_count = read(read_fd, buf, COPY_BUFSZ);
    if (read_count == -1) die(1, "event thread: error reading", "");

    event_len = (int) ntohs(*((uint16_t *) buf));

    if (debug) syslog(LOG_DEBUG, "read %d bytes from connection %ld",
                      read_count, connection->id);

    if (read_count != event_len) {
      syslog(LOG_ERR, "event thread: only read %d of %d",
             read_count, event_len);

      msg = must_malloc("event hex", read_count * 2 + 1);
      for (int i = 0; i < read_count; i++) {
        sprintf(((char *) msg) + (i * 2),"%02x",(int) (((char *) buf)[i]));
      }
      ((char *) msg)[read_count * 2] = 0;
      syslog(LOG_ERR, "message: %s", (char *) msg);
      free(msg);

      continue;
    }

    path_len  = (int) ntohs(*(((uint16_t *) buf) + 1));
    // TODO: could check the path length isn't a lie here
    path = (char *) (((uint8_t *) buf) + 4);
    // TODO: could check the path is NUL terminated here
    syscall = *(((uint8_t *) buf) + 4 + path_len);

    // TODO: should this be in another thread?
    perform_syscall(syscall, path);
  }

  close(read_fd);
  free(buf);
  free(read_path);
  return NULL;
}

void toggle_debug(int sig) {
  debug = !debug;
}

void setup_debug() {
  if (SIG_ERR == signal(SIGHUP, toggle_debug))
    die(1, "Couldn't set SIGHUP behavior", "");

  if (siginterrupt(SIGHUP, 1))
    die(1, "Couldn't set siginterrupt for SIGHUP", "");
}

void parse_parameters(int argc, char * argv[], parameters * params) {
  int c;
  int errflg = 0;

  params->pidfile = NULL;
  params->socket9p_root = NULL;
  params->fusermount = NULL;

  while ((c = getopt(argc, argv, ":p:9:f:")) != -1) {
    switch(c) {

    case 'p':
      params->pidfile = optarg;
      break;

    case '9':
      params->socket9p_root = optarg;
      break;

    case 'f':
      params->fusermount = optarg;
      break;

    case ':':
      fprintf(stderr, "Option -%c requires a path argument\n", optopt);
      errflg++;
      break;

    case '?':
      fprintf(stderr, "Unrecognized option: '-%c'\n", optopt);
      errflg++;
      break;

    default:
      fprintf(stderr, "Internal error parsing -%c\n", c);
      errflg++;
    }
  }

  if (errflg) die(2, NULL, usage);

  if (params->pidfile != NULL && access(params->pidfile, W_OK))
    if (errno != ENOENT)
      die(2, "", "-p %s path to pidfile must be writable: ", params->pidfile);

  if (params->fusermount == NULL)
    params->fusermount = default_fusermount;
  if (access(params->fusermount, X_OK))
    die(2, "", "-f %s path to fusermount must be executable: ",
        params->fusermount);

  if (params->socket9p_root == NULL)
    params->socket9p_root = default_socket9p_root;
  if (access(params->socket9p_root, X_OK))
    die(2, "", "-9 %s path to socket 9p root directory must be executable: ",
        params->socket9p_root);
}

void write_pidfile(char * pidfile) {
  int fd;
  pid_t pid = getpid();
  char * pid_s;
  int pid_s_len, write_count;

  if (asprintf(&pid_s, "%lld", (long long) pid) == -1)
    die(1, "Couldn't allocate pidfile string", "");

  pid_s_len = strlen(pid_s);

  fd = open(pidfile, O_WRONLY | O_CREAT | O_TRUNC, 0644);
  if (fd == -1)
    die(1, "", "Couldn't open pidfile path %s: ", pidfile);

  write_count = write(fd, pid_s, pid_s_len);
  if (write_count == -1)
    die(1, "", "Error writing pidfile %s: ", pidfile);

  if (write_count != pid_s_len)
    die(1, NULL, "Error writing %s to pidfile %s: only wrote %d bytes\n",
        pid_s, pidfile, write_count);

  close(fd);
  free(pid_s);
}

#define ID_LEN 512

void process_events(char * events_path, int events, parameters * params) {
    char buf[ID_LEN];
    int read_count;
    long conn_id;
    pthread_t child;
    connection_state * conn;
    char * connection_type;
    void * (*connection_handler_thread)(void *);

    while (1) {
      read_count = read(events, buf, ID_LEN - 1);
      if (read_count == -1) {
        die(1, "", "Error reading events path %s: ", events_path);
      } else if (read_count == 0) {
        // TODO: this is probably the 9p server's fault due to
        //       not dropping the read 0 to force short read if
        //       the real read is flushed
        syslog(LOG_WARNING, "read 0 from event stream %s", events_path);
        continue;
      }

      buf[read_count] = 0x0;

      if (read_count < 2) {
        die(1, NULL, "Event connection id isn't long enough");
      }

      errno = 0;
      conn_id = strtol(buf + 1, NULL, 10);
      if (errno) die(1, "failed", "Connection id of string '%s'", buf);

      if (debug) syslog(LOG_DEBUG, "handle connection %ld", conn_id);

      conn = (connection_state *) must_malloc("connection state",
                                              sizeof(connection_state));
      conn->id = conn_id;
      conn->params = params;

      switch (buf[0]) {
      case 'm':
        connection_type = "mount";
        connection_handler_thread = mount_thread;
        break;
      case 'e':
        connection_type = "event";
        connection_handler_thread = event_thread;
        break;
      default:
        die(1, NULL, "Unknown connection type '%c'", buf[0]);
      }

      if ((errno = pthread_create(&child, NULL,
                                  connection_handler_thread, conn)))
        die(1, "", "Couldn't create thread for %s connection '%ld': ",
            connection_type, conn_id);

      if ((errno = pthread_detach(child)))
        die(1, "", "Couldn't detach thread for %s connection '%ld': ",
            connection_type, conn_id);

      if (debug) syslog(LOG_DEBUG, "thread spawned");
    }
}

int main(int argc, char * argv[]) {
  int events;
  parameters params;
  char * events_path;

  openlog(argv[0], LOG_CONS, LOG_DAEMON);

  parse_parameters(argc, argv, &params);
  setup_debug();

  if (params.pidfile != NULL) write_pidfile(params.pidfile);

  if (asprintf(&events_path, "%s/events", params.socket9p_root) == -1)
    die(1, "Couldn't allocate events path", "");

  events = open(events_path, O_RDONLY | O_CLOEXEC);
  if (events != -1) process_events(events_path, events, &params);

  syslog(LOG_CRIT, "Failed to open events path %s: %s",
         events_path, strerror(errno));
  free(events_path);
  return 1;
}
