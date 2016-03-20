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

#define COPY_BUFSZ 65536
// The Linux 9p driver/xhyve virtio-9p device have bugs in the
// zero-copy code path which is triggered by I/O of more than 1024
// bytes. For an unknown reason, these defects are unusually prominent
// in the event channel use pattern.
#define EVENT_BUFSZ 1024

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
  "                  [-l logfile] [-m mount_trigger] [-t triggerlog]\n"
  " -p pidfile\tthe path at which to write the pid of the process\n"
  " -9 " DEFAULT_SOCKET9P_ROOT "\tthe root of the 9p socket file system\n"
  " -f " DEFAULT_FUSERMOUNT "\tthe fusermount executable to use\n"
  " -l logfile\tthe log file to use before the mount trigger\n"
  " -m mount_trigger\tthe mountpoint to use to trigger log switchover\n"
  " -t triggerlog\tthe file to use after the trigger\n";

int debug = 0;

pthread_attr_t detached;

typedef struct {
  char * descr;
  long connection;
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

char ** read_opts(connection_t * connection, char * buf) {
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
        die(1, NULL, "copy %s trace: read %d but only wrote %d",
            descr, read_count, write_count);

      close(trace_fd);
      free(trace_path);
    }

    write_count = write(to, buf, read_count);
    if (write_count == -1) die(1, "", "copy %s: error writing: ", descr);

    if (write_count != read_count)
      die(1, NULL, "copy %s: read %d but only wrote %d",
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

void * copy_clean_to_thread(void * copy_state) {
  return (copy_clean_to((copy_thread_state *) copy_state));
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
  if ((errno = pthread_create(&child, &detached,
                              copy_clean_from_thread, copy_state)))
    die(1, "", "couldn't create read copy thread for connection %ld: ",
        connection->id);
}

void start_writer(connection_t * connection, int fuse) {
  int write_fd;
  char * write_path;
  pthread_t child;
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
  if ((errno = pthread_create(&child, &detached,
                              copy_clean_to_thread, copy_state)))
    die(1, "", "Couldn't create write copy thread for connection %ld: ",
        connection->id);
}

void * mount_connection(connection_t * conn) {
  int optc;
  char ** optv;
  int fuse;
  char * buf;
  pthread_mutex_t copy_lock;
  pthread_cond_t copy_halt;
  int should_halt = 0;
  
  buf = (char *) must_malloc("read_opts packet malloc", COPY_BUFSZ);

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
      die(1, "", "Couldn't wait for copy halt for connection %ld: ",
          conn->id);
  unlock("copy lock", &copy_lock);

  free(conn);

  return NULL;
}

void * mount_thread(void * connection) {
  return mount_connection((connection_t *) connection);
}

void write_pid(connection_t * connection) {
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
    die(1, NULL, "Error writing pid %s to socket: only wrote %d bytes",
        pid_s, write_count);

  close(write_fd);
  free(pid_s);
  free(write_path);
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

  default:
    die(1, NULL, "Unknown event syscall %" PRIu8, syscall);
  }

  if (r != 0)
    thread_log_time(conn, "Event %s %s error: %s\n",
                    name, path, strerror(errno));
}

void * event_thread(void * connection_ptr) {
  char * read_path;
  int read_fd;
  int read_count, event_len, path_len;
  void * buf;
  connection_t * connection = connection_ptr;

  char * path;
  uint8_t syscall;
  void * msg;

  // This thread registers with the mounted file system as being an
  // fsnotify event actuator. Other mounted file system interactions
  // (such as self-logging) SHOULD NOT occur on this thread.

  write_pid(connection);

  if (asprintf(&read_path, "%s/connections/%ld/read",
               connection->params->socket9p_root, connection->id) == -1)
    die(1, "Couldn't allocate read path", "");

  read_fd = open(read_path, O_RDONLY);
  if (read_fd == -1)
    die(1, "couldn't open read path", "For connection %ld, ", connection->id);

  buf = must_malloc("incoming event buffer", EVENT_BUFSZ);

  while(1) {
    read_count = read(read_fd, buf, EVENT_BUFSZ);
    if (read_count == -1) die(1, "event thread: error reading", "");

    event_len = (int) ntohs(*((uint16_t *) buf));

    if (debug)
      thread_log_time(connection, "read %d bytes from connection %ld\n",
                      read_count, connection->id);

    if (read_count != event_len) {
      thread_log_time(connection, "event thread: only read %d of %d\n",
                      read_count, event_len);

      msg = must_malloc("event hex", read_count * 2 + 1);
      for (int i = 0; i < read_count; i++) {
        sprintf(((char *) msg) + (i * 2),"%02x",(int) (((char *) buf)[i]));
      }
      ((char *) msg)[read_count * 2] = 0;
      thread_log_time(connection, "message: %s\n", (char *) msg);
      free(msg);

      continue;
    }

    path_len  = (int) ntohs(*(((uint16_t *) buf) + 1));
    // TODO: could check the path length isn't a lie here
    path = (char *) (((uint8_t *) buf) + 4);
    // TODO: could check the path is NUL terminated here
    syscall = *(((uint8_t *) buf) + 4 + path_len);

    // TODO: should this be in another thread?
    perform_syscall(connection, syscall, path);
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
  params->logfile = NULL;
  params->logfile_fd = 0;
  params->mount_trigger = NULL;
  params->trigger_log = NULL;
  params->trigger_fd = 0;
  lock_init("fd_lock", &params->fd_lock, NULL);

  while ((c = getopt(argc, argv, ":p:9:f:l:m:t:")) != -1) {
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

  if (params->socket9p_root == NULL)
    params->socket9p_root = default_socket9p_root;
  if (access(params->socket9p_root, X_OK)) {
    fprintf(stderr,
            "-9 %s path to socket 9p root directory must be executable: ",
            params->socket9p_root);
    perror("");
    exit(2);
  }

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

#define ID_LEN 512

void process_events(char * events_path, int events, parameters * params) {
    char buf[ID_LEN];
    int read_count;
    long conn_id;
    pthread_t child;
    connection_t * conn;
    void * (*connection_handler_thread)(void *);

    while (1) {
      conn = (connection_t *) must_malloc("connection state",
                                              sizeof(connection_t));
      conn->params = params;
      conn->id = 0;

      read_count = read(events, buf, ID_LEN - 1);
      if (read_count == -1) {
        die(1, "", "Error reading events path %s: ", events_path);
      } else if (read_count == 0) {
        // TODO: this is probably the 9p server's fault due to
        //       not dropping the read 0 to force short read if
        //       the real read is flushed
        log_time(conn, "read 0 from event stream %s\n", events_path);
        continue;
      }

      buf[read_count] = 0x0;

      if (read_count < 2) {
        die(1, NULL, "Event connection id isn't long enough");
      }

      errno = 0;
      conn_id = strtol(buf + 1, NULL, 10);
      if (errno) die(1, "failed", "Connection id of string '%s'", buf);

      conn->id = conn_id;

      if (debug) log_time(conn, "handle connection %ld\n", conn_id);

      switch (buf[0]) {
      case 'm':
        conn->type_descr = "mount";
        connection_handler_thread = mount_thread;
        break;
      case 'e':
        conn->type_descr = "event";
        connection_handler_thread = event_thread;
        break;
      default:
        die(1, NULL, "Unknown connection type '%c'", buf[0]);
      }

      if ((errno = pthread_create(&child, &detached,
                                  connection_handler_thread, conn)))
        die(1, "", "Couldn't create thread for %s connection '%ld': ",
            conn->type_descr, conn_id);

      if (debug) log_time(conn, "thread spawned\n");
    }
}

int main(int argc, char * argv[]) {
  int events;
  parameters params;
  char * events_path;
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

  if (params.pidfile != NULL) write_pidfile(params.pidfile);

  if (params.logfile != NULL) {
    params.logfile_fd = open(params.logfile, O_WRONLY | O_APPEND | O_CREAT);
    if (params.logfile_fd == -1)
      die(1, "", "Couldn't open log file %s: ", params.logfile);
  }

  if (asprintf(&events_path, "%s/events", params.socket9p_root) == -1)
    die(1, "Couldn't allocate events path", "");

  events = open(events_path, O_RDONLY | O_CLOEXEC);
  if (events != -1) process_events(events_path, events, &params);

  connection_t top;
  top.params = &params;
  top.id = 0;
  log_time(&top, "Failed to open events path %s: %s\n",
           events_path, strerror(errno));
  free(events_path);
  return 1;
}
