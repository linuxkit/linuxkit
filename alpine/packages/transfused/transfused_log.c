#include <errno.h>
#include <string.h>

#include <stdlib.h>
#include <stdarg.h>
#include <stdio.h>
#include <pthread.h>

#include <syslog.h>

#include <sys/time.h>
#include <time.h>
#include <math.h>

#include "transfused.h"

void log_timestamp(int fd) {
  char timestamp[26];
  int msec;
  struct tm* tm_info;
  struct timeval tv;

  gettimeofday(&tv, NULL);

  msec = lrint(tv.tv_usec/1000.0);
  if (msec >= 1000) {
    msec -= 1000;
    tv.tv_sec++;
  }

  tm_info = localtime(&tv.tv_sec);

  strftime(timestamp, 26, "%Y-%m-%d %H:%M:%S", tm_info);
  dprintf(fd, "%s.%03d ", timestamp, msec);
}

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

void vlog_locked(connection_t * conn, const char * fmt, va_list args) {
  int fd = conn->params->trigger_fd;
  if (fd != 0) {
    vdprintf(fd, fmt, args);
  } else {
    vsyslog(LOG_INFO, fmt, args);
    fd = conn->params->logfile_fd;
    if (fd != 0) {
      vdprintf(fd, fmt, args);
    }
  }  
}

void vlog_time_locked(connection_t * conn, const char * fmt, va_list args) {
  int fd = conn->params->trigger_fd;
  if (fd != 0) log_timestamp(fd);
  else {
    fd = conn->params->logfile_fd;
    if (fd != 0) log_timestamp(fd);
  }
  vlog_locked(conn, fmt, args);
}

void log_time_locked(connection_t * connection, const char * fmt, ...) {
  va_list args;

  va_start(args, fmt);

  vlog_time_locked(connection, fmt, args);

  va_end(args);
}

void log_time(connection_t * connection, const char * fmt, ...) {
  va_list args;

  va_start(args, fmt);

  lock("log_time fd_lock", &connection->params->fd_lock);
  vlog_time_locked(connection, fmt, args);
  unlock("log_time fd_lock", &connection->params->fd_lock);

  va_end(args);
}

typedef struct {
  connection_t * connection;
  char * msg;
} log_thread_state;

void * log_time_thread(void * log_state_ptr) {
  log_thread_state * log_state = log_state_ptr;

  log_time(log_state->connection, log_state->msg);

  free(log_state->msg);
  free(log_state);

  return NULL;
}

void thread_log_time(connection_t * conn, const char * fmt, ...) {
  pthread_t logger;
  va_list args;
  log_thread_state * log_state;

  log_state = must_malloc("thread_log_time log_state",
                          sizeof(log_thread_state));
  log_state->connection = conn;

  va_start(args, fmt);
  if (vasprintf(&log_state->msg, fmt, args) == -1)
    die(1, "Couldn't allocate thread_log_time message", "");
  va_end(args);

  // TODO: We currently spawn a new thread for every message. This is
  // far from ideal but fine for now as we anticipate thread-sensitive
  // log demand to be low.

  if ((errno = pthread_create(&logger, &detached, log_time_thread, log_state)))
    die(1, "", "Couldn't create log thread for %s connection '%ld': ",
        conn->type_descr, conn->id);
}

void log_continue_locked(connection_t * connection, const char * fmt, ...) {
  va_list args;

  va_start(args, fmt);

  vlog_locked(connection, fmt, args);

  va_end(args);
}

void log_continue(connection_t * connection, const char * fmt, ...) {
  va_list args;

  va_start(args, fmt);

  lock("log_continue fd_lock", &connection->params->fd_lock);
  vlog_locked(connection, fmt, args);
  unlock("log_continue fd_lock", &connection->params->fd_lock);

  va_end(args);
}
