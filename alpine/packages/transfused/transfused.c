#include <sys/types.h>
#include <errno.h>

#include <stdlib.h>
#include <string.h>
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

#include <sys/time.h>
#include <sys/resource.h>

#include "transfused_log.h"
#include "transfused_vsock.h"

#define IN_BUFSZ  ((1 << 20) + 16)
#define OUT_BUFSZ ((1 << 20) + 64)
#define EVENT_BUFSZ 4096

#define DEFAULT_FUSERMOUNT "/bin/fusermount"
#define DEFAULT_SOCKET "v:_:1525"
#define DEFAULT_SERVER "v:2:1524"

#define RMDIR_SYSCALL    0
#define UNLINK_SYSCALL   1
#define MKDIR_SYSCALL    2
#define SYMLINK_SYSCALL  3
#define TRUNCATE_SYSCALL 4
#define CHMOD_SYSCALL    5
// these could be turned into an enum probably but... C standard nausea

char * default_fusermount = DEFAULT_FUSERMOUNT;
char * default_socket = DEFAULT_SOCKET;
char * default_server = DEFAULT_SERVER;
char * usage =
  "usage: transfused [-p pidfile] [-d server] [-s socket] [-f fusermount]\n"
  "                  [-l logfile] [-m mount_trigger] [-t triggerlog]\n"
  " -p pidfile\tthe path at which to write the pid of the process\n"
  " -d " DEFAULT_SERVER "\tthe server address to use ('v:addr:port')\n"
  " -s " DEFAULT_SOCKET "\tthe socket address to use ('v:addr:port')\n"
  " -f " DEFAULT_FUSERMOUNT "\tthe fusermount executable to use\n"
  " -l logfile\tthe log file to use before the mount trigger\n"
  " -m mount_trigger\tthe mountpoint to use to trigger log switchover\n"
  " -t triggerlog\tthe file to use after the trigger\n";

int debug = 0;

pthread_attr_t detached;

typedef struct {
  char * descr;
  char * tag;
  int from;
  int to;
} copy_thread_state;

#include <sys/syscall.h>

pid_t gettid() {
  return syscall(SYS_gettid);
}

void * must_malloc(char *const descr, size_t size) {
  void * ptr;

  ptr = malloc(size);
  if (size != 0 && ptr == NULL) die(1, descr, "");

  return ptr;
}

void cond_init(char *const descr, pthread_cond_t * cond,
               const pthread_condattr_t *restrict attr) {
  if ((errno = pthread_cond_init(cond, attr)))
    die(1, "", "cond init %s: ", descr);
}

void lock_init(char *const descr, pthread_mutex_t * mutex,
               const pthread_mutexattr_t *restrict attr) {
  if ((errno = pthread_mutex_init(mutex, attr)))
    die(1, "", "lock init %s: ", descr);
}

void lock(char *const descr, pthread_mutex_t * mutex) {
  if ((errno = pthread_mutex_lock(mutex)))
    die(1, "", "lock %s: ", descr);
}

void unlock(char *const descr, pthread_mutex_t * mutex) {
  if ((errno = pthread_mutex_unlock(mutex)))
    die(1, "", "unlock %s: ", descr);
}

int bind_socket(const char * socket) {
  int sock;

  if (socket[0] == 0)
    die(2, NULL, "Socket family required");
  if (socket[1] != ':')
    die(2, NULL, "Socket address required");

  switch (socket[0]) {
  case 'v':
    sock = bind_vsock(socket + 2);
    break;
  default:
    die(2, NULL, "Unknown socket family '%c'", socket[0]);
  }

  return sock;
}

int connect_socket(const char * socket) {
  int sock;

  if (socket[0] == 0)
    die(2, NULL, "Socket family required");
  if (socket[1] != ':')
    die(2, NULL, "Scoket address required");

  switch (socket[0]) {
  case 'v':
    sock = connect_vsock(socket + 2);
    break;
  default:
    die(2, NULL, "Unknown socket family '%c'", socket[0]);
  }

  return sock;
}

char ** read_opts(connection_t * connection, char * buf) {
  int read_count;
  int optc = 1;
  char ** optv;
  size_t mount_len;

  // TODO: deal with socket read conditions e.g. EAGAIN
  read_count = read(connection->sock, buf, EVENT_BUFSZ - 1);
  if (read_count < 0) die(1, "read_opts error reading", "");

  // TODO: protocol should deal with short read
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

  mount_len = strnlen(optv[optc - 1], 4096) + 1;
  connection->mount_point = must_malloc("mount point string", mount_len);
  strncpy(connection->mount_point, optv[optc - 1], mount_len - 1);
  connection->mount_point[mount_len - 1] = '\0';

  return optv;
}

uint64_t message_id(uint64_t * message) {
  return message[1];
}

int read_message(char * descr, int fd, char * buf, size_t max_read) {
  int read_count;
  size_t nbyte;
  uint32_t len;

  // TODO: socket read conditions e.g. EAGAIN
  read_count = read(fd, buf, 4);
  if (read_count != 4) {
    if (read_count < 0) die(1, "", "read %s: error reading: ", descr);
    if (read_count == 0) die(1, NULL, "read %s: EOF reading length", descr);
    die(1, NULL, "read %s: short read length %d", descr, read_count);
  }
  len = *((uint32_t *) buf);
  if (len > max_read)
    die(1, NULL, "read %s: message size %d exceeds buffer capacity %d",
        len, max_read);

  nbyte = (size_t) (len - 4);
  buf += 4;

  do {
    // TODO: socket read conditions e.g. EAGAIN
    read_count = read(fd, buf, nbyte);
    if (read_count < 0) die(1, "", "read %s: error reading: ", descr);
    if (read_count == 0) die(1, NULL, "read %s: EOF reading", descr);
    nbyte -= read_count;
    buf += read_count;
  } while (nbyte != 0);

  return (int) len;
}

void copy_into_fuse(copy_thread_state * copy_state) {
  int from = copy_state->from;
  int to = copy_state->to;
  char * descr = copy_state->descr;
  int read_count, write_count;
  void * buf;

  buf = must_malloc(descr, IN_BUFSZ);

  while(1) {
    read_count = read_message(descr, from, (char *) buf, IN_BUFSZ);

    write_count = write(to, buf, read_count);
    if (write_count < 0) die(1, "", "copy %s: error writing: ", descr);

    // /dev/fuse accepts only complete writes
    if (write_count != read_count)
      die(1, NULL, "copy %s: read %d but only wrote %d",
          descr, read_count, write_count);
  }

  free(buf);
}

void write_exactly(char * descr, int fd, char * buf, size_t nbyte) {
  int write_count;

  do {
    // TODO: socket write conditions e.g. EAGAIN
    write_count = write(fd, buf, nbyte);
    if (write_count < 0) die(1, "", "copy %s: error writing: ", descr);
    if (write_count == 0) die(1, "", "copy %s: 0 write: ", descr);

    nbyte -= write_count;
    buf += write_count;
  } while (nbyte != 0);
}

void copy_outof_fuse(copy_thread_state * copy_state) {
  int from = copy_state->from;
  int to = copy_state->to;
  char * descr = copy_state->descr;
  int read_count;
  void * buf;

  buf = must_malloc(descr, OUT_BUFSZ);

  while(1) {
    // /dev/fuse only returns complete reads
    read_count = read(from, buf, OUT_BUFSZ);
    if (read_count < 0) die(1, "", "copy %s: error reading: ", descr);

    write_exactly(descr, to, (char *) buf, read_count);
  }

  free(buf);
}

void * copy_clean_into_fuse(copy_thread_state * copy_state) {
  copy_into_fuse(copy_state);

  close(copy_state->from);

  free(copy_state->descr);
  free(copy_state);

  return NULL;
}

void * copy_clean_into_fuse_thread(void * copy_state) {
  return (copy_clean_into_fuse((copy_thread_state *) copy_state));
}

void * copy_clean_outof_fuse(copy_thread_state * copy_state) {
  copy_outof_fuse(copy_state);

  close(copy_state->to);

  free(copy_state->descr);
  free(copy_state);

  return NULL;
}

void * copy_clean_outof_fuse_thread(void * copy_state) {
  return (copy_clean_outof_fuse((copy_thread_state *) copy_state));
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
int get_fuse_sock(connection_t * conn, int optc, char *const optv[]) {
  char ** argv;
  char * envp[2];
  pid_t fusermount_pid;
  int fuse_socks[2];
  int status;
  int fd;

  // prepare argv from optv
  argv = (char **) must_malloc("fusermount argv", (optc + 2) * sizeof(char *));

  argv[0] = conn->params->fusermount;
  memcpy(&argv[1], optv, (optc + 1) * sizeof(char *));

  lock("get_fuse_sock fd_lock", &conn->params->fd_lock);
  log_time_locked(conn,"mount ");
  for (int i = 0; argv[i]; i++) log_continue_locked(conn, "%s ",argv[i]);
  log_continue_locked(conn, "\n");
  unlock("get_fuse_sock fd_lock", &conn->params->fd_lock);

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
    if (execve(argv[0], argv, envp))
      die(1, "Failed to execute fusermount", "");

  // parent
  free(argv);
  free(envp[0]);

  // close the end of the socket that we gave away
  close(fuse_socks[0]);

  // wait for fusermount to return
  waitpid(fusermount_pid, &status, 0);
  if (!WIFEXITED(status))
    die(1, NULL, "fusermount terminated abnormally");

  if (WEXITSTATUS(status))
    die(1, NULL, "fusermount exited with code %d", WEXITSTATUS(status));

  if (debug) log_time(conn, "about to recv_fd from fusermount\n");

  fd = recv_fd(fuse_socks[1]);
  if (fd == -1)
    die(1, NULL, "Couldn't receive fd over FUSE socket");

  // close the read end of the socket
  close(fuse_socks[1]);

  return fd;
}

void start_reader(connection_t * connection, int fuse) {
  pthread_t child;
  copy_thread_state * copy_state;

  copy_state = (copy_thread_state *) must_malloc("start_reader copy_state",
                                                 sizeof(copy_thread_state));
  copy_state->descr = connection->mount_point;
  copy_state->tag = "read";
  copy_state->from = connection->sock;
  copy_state->to = fuse;
  if ((errno = pthread_create(&child, &detached,
                              copy_clean_into_fuse_thread, copy_state)))
    die(1, "", "couldn't create read copy thread for mount %s: ",
        connection->mount_point);
}

void start_writer(connection_t * connection, int fuse) {
  pthread_t child;
  copy_thread_state * copy_state;

  copy_state = (copy_thread_state *) must_malloc("do_write copy_state",
                                                 sizeof(copy_thread_state));
  copy_state->descr = connection->mount_point;
  copy_state->tag = "write";
  copy_state->from = fuse;
  copy_state->to = connection->sock;
  if ((errno = pthread_create(&child, &detached,
                              copy_clean_outof_fuse_thread, copy_state)))
    die(1, "", "Couldn't create write copy thread for mount %s: ",
        connection->mount_point);
}

void * mount_connection(connection_t * conn) {
  int optc;
  char ** optv;
  int fuse;
  char * buf;
  pthread_mutex_t copy_lock;
  pthread_cond_t copy_halt;
  int should_halt = 0;
  
  buf = (char *) must_malloc("read_opts packet malloc", EVENT_BUFSZ);

  optv = read_opts(conn, buf);

  for (optc = 0; optv[optc] != NULL; optc++) {}

  fuse = get_fuse_sock(conn, optc, optv);
  free(buf);

  lock_init("copy_lock", &copy_lock, NULL);
  cond_init("copy_halt", &copy_halt, NULL);

  start_reader(conn, fuse);
  start_writer(conn, fuse);

  // trigger?
  // TODO: strcmp scares me
  // TODO: append logfile to trigger_log
  if (conn->params->mount_trigger != NULL
      && conn->params->trigger_log != NULL
      && 0 == strcmp(optv[optc - 1], conn->params->mount_trigger)) {

    lock("trigger mount fd_lock", &conn->params->fd_lock);
    log_time_locked(conn, "Log mount trigger fired on %s, logging to %s\n",
                    conn->params->mount_trigger, conn->params->trigger_log);
    conn->params->trigger_fd = open(conn->params->trigger_log,
                                    O_WRONLY | O_APPEND | O_CREAT, 0600);
    if (conn->params->trigger_fd == -1)
      die(1, "", "Couldn't open trigger log %s: ", conn->params->trigger_log);
    unlock("trigger mount fd_lock", &conn->params->fd_lock);
  }

  free(optv);

  lock("copy lock", &copy_lock);
  while (!should_halt)
    if ((errno = pthread_cond_wait(&copy_halt, &copy_lock)))
      die(1, "", "Couldn't wait for copy halt for mount %s: ",
          conn->mount_point);
  unlock("copy lock", &copy_lock);

  free(conn);

  return NULL;
}

void * mount_thread(void * connection) {
  return mount_connection((connection_t *) connection);
}

void write_pid(connection_t * connection) {
  pid_t pid = gettid();
  char * pid_s;
  int pid_s_len, write_count;

  if (asprintf(&pid_s, "%lld", (long long) pid) == -1)
    die(1, "Couldn't allocate pid string", "");

  pid_s_len = strlen(pid_s);

  // TODO: check for socket write conditions e.g. EAGAIN
  write_count = write(connection->sock, pid_s, pid_s_len);
  if (write_count < 0)
    die(1, "Error writing pid", "");

  // TODO: handle short writes
  if (write_count != pid_s_len)
    die(1, NULL, "Error writing pid %s to socket: only wrote %d bytes",
        pid_s, write_count);

  free(pid_s);
}

void perform_syscall(connection_t * conn, uint8_t syscall, char path[]) {
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

  case CHMOD_SYSCALL:
    name = "chmod";
    r = chmod(path, 0700);
    break;

  default:
    die(1, NULL, "Unknown event syscall %" PRIu8, syscall);
  }

  if (r != 0)
    thread_log_time(conn, "Event %s %s error: %s\n",
                    name, path, strerror(errno));
}

void * event_thread(void * connection_ptr) {
  int read_count, path_len;
  void * buf;
  connection_t * connection = connection_ptr;

  char * path;
  uint8_t syscall;

  // This thread registers with the file system server as being an
  // fsnotify event actuator. Other mounted file system interactions
  // (such as self-logging) SHOULD NOT occur on this thread.

  write_pid(connection);

  buf = must_malloc("incoming event buffer", EVENT_BUFSZ);

  while(1) {
    read_count = read_message("events", connection->sock, buf, EVENT_BUFSZ);

    if (debug)
      thread_log_time(connection, "read %d bytes from event connection\n",
                      read_count);

    path_len  = (int) ntohs(*(((uint32_t *) buf) + 1));
    // TODO: could check the path length isn't a lie here
    path = (char *) (((uint8_t *) buf) + 6);
    // TODO: could check the path is NUL terminated here
    syscall = *(((uint8_t *) buf) + 6 + path_len);

    // TODO: should this be in another thread?
    perform_syscall(connection, syscall, path);
  }

  free(buf);
  // TODO: close connection
  return NULL;
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
    die(1, NULL, "Error writing %s to pidfile %s: only wrote %d bytes",
        pid_s, pidfile, write_count);

  close(fd);
  free(pid_s);
}

void * init_thread(void * params_ptr) {
  parameters * params = params_ptr;
  int init_sock = connect_socket(params->server);
  int write_count, read_count;
  char init_msg[6] = { '\6', '\0', '\0', '\0', '\0', '\0' };
  void * buf;

  // TODO: handle short write/socket conditions
  write_count = write(init_sock, init_msg, sizeof(init_msg));
  if (write_count < 0) die(1, "init thread: couldn't write init", "");
  if (write_count != sizeof(init_msg))
    die(1, "init thread: incomplete write", "");

  buf = must_malloc("incoming init buffer", EVENT_BUFSZ);

  // TODO: handle short read/socket conditions
  read_count = read(init_sock, buf, EVENT_BUFSZ);
  if (read_count < 0) die(1, "init thread: error reading", "");
  // TODO: handle other messages
  if (read_count != 6) die(1, "init thread: response not 6", "");
  for (int i = 0; i < sizeof(init_msg); i++)
    if (((char *)buf)[i] != init_msg[i])
      die(1, "init thread: unexpected message", "");

  // we've gotten Continue so write the pidfile
  if (params->pidfile != NULL)
    write_pidfile(params->pidfile);

  // TODO: handle more messages
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
  params->socket = NULL;
  params->fusermount = NULL;
  params->logfile = NULL;
  params->logfile_fd = 0;
  params->mount_trigger = NULL;
  params->trigger_log = NULL;
  params->trigger_fd = 0;
  lock_init("fd_lock", &params->fd_lock, NULL);

  while ((c = getopt(argc, argv, ":p:d:s:f:l:m:t:")) != -1) {
    switch(c) {

    case 'p':
      params->pidfile = optarg;
      break;

    case 'd':
      params->server = optarg;
      break;

    case 's':
      params->socket = optarg;
      break;

    case 'f':
      params->fusermount = optarg;
      break;

    case 'l':
      params->logfile = optarg;
      break;

    case 'm':
      params->mount_trigger = optarg;
      break;

    case 't':
      params->trigger_log = optarg;
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

  if (errflg) {
    fprintf(stderr, "%s", usage);
    exit(2);
  }

  if (params->pidfile != NULL && access(params->pidfile, W_OK))
    if (errno != ENOENT) {
      fprintf(stderr, "-p %s path to pidfile must be writable: ",
              params->pidfile);
      perror("");
      exit(2);
    }

  if (params->fusermount == NULL)
    params->fusermount = default_fusermount;
  if (access(params->fusermount, X_OK)) {
    fprintf(stderr, "-f %s path to fusermount must be executable: ",
            params->fusermount);
    perror("");
    exit(2);
  }

  if (params->socket == NULL)
    params->socket = default_socket;

  if (params->server == NULL)
    params->server = default_server;

  if (params->logfile != NULL && access(params->logfile, W_OK))
    if (errno != ENOENT) {
      fprintf(stderr, "-l %s path to logfile must be writable: ",
              params->logfile);
      perror("");
      exit(2);
    }

  if (params->mount_trigger != NULL
      && access(params->mount_trigger, F_OK)) {
    fprintf(stderr, "-m %s path to mount point must exist: ",
            params->mount_trigger);
    perror("");
    exit(2);
  }
}

void serve(parameters * params) {
  ssize_t read_count;
  char subproto_selector;
  pthread_t child;
  connection_t * conn;
  void * (*connection_handler_thread)(void *);

  if (listen(params->sock, 16))
    die(1, "listen", "");

  if ((errno = pthread_create(&child, &detached, init_thread, params)))
    die(1, "", "Couldn't create initialization thread: ");

  while (1) {
    conn = (connection_t *) must_malloc("connection state",
                                        sizeof(connection_t));
    conn->params = params;
    conn->mount_point = "";

    conn->sock = accept(params->sock, &conn->sa_client, &conn->socklen_client);
    if (conn->sock < 0)
      die(1, "accept", "");

    // TODO: check for socket read conditions e.g. EAGAIN
    read_count = read(conn->sock, &subproto_selector, 1);
    if (read_count <= 0)
      die(1, "read subprotocol selector", "");

    switch (subproto_selector) {
    case 'm':
      conn->type_descr = "mount";
      connection_handler_thread = mount_thread;
      break;
    case 'e':
      conn->type_descr = "event";
      connection_handler_thread = event_thread;
      break;
    default:
      die(1, NULL, "Unknown subprotocol type '%c'", subproto_selector);
    }

    if ((errno = pthread_create(&child, &detached,
                                connection_handler_thread, conn)))
      die(1, "", "Couldn't create thread for %s connection: ",
          conn->type_descr);

    if (debug) log_time(conn, "thread spawned\n");
  }
}

int main(int argc, char * argv[]) {
  parameters params;
  struct rlimit core_limit;

  core_limit.rlim_cur = RLIM_INFINITY;
  core_limit.rlim_max = RLIM_INFINITY;
  if (setrlimit(RLIMIT_CORE, &core_limit))
    die(1, "", "Couldn't set RLIMIT_CORE to RLIM_INFINITY");

  openlog(argv[0], LOG_CONS | LOG_PERROR | LOG_NDELAY, LOG_DAEMON);

  parse_parameters(argc, argv, &params);
  setup_debug();

  if ((errno = pthread_attr_setdetachstate(&detached,
                                           PTHREAD_CREATE_DETACHED)))
    die(1, "Couldn't set pthread detach state", "");

  if (params.logfile != NULL) {
    params.logfile_fd = open(params.logfile, O_WRONLY | O_APPEND | O_CREAT);
    if (params.logfile_fd == -1)
      die(1, "", "Couldn't open log file %s: ", params.logfile);
  }

  params.sock = bind_socket(params.socket);
  serve(&params);

  return 0;
}
