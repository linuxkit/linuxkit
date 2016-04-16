#include <errno.h>
#include <string.h>

#include <stdlib.h>
#include <stdarg.h>
#include <stdio.h>
#include <pthread.h>
#include <unistd.h>

#include <syslog.h>

#include <sys/time.h>
#include <time.h>
#include <math.h>
#include <inttypes.h>

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

void die
(int exit_code, parameters * params, const char * parg, const char * fmt, ...);

void vlog_sock_locked(int fd, const char * fmt, va_list args) {
  uint16_t log_err_type = 1;
  int rc, len;
  va_list targs;
  char * fill;

  va_copy(targs, args);
  len = vsnprintf(NULL, 0, fmt, targs);
  if (len < 0) die(1, NULL, NULL, "Couldn't log due to vsnprintf failure");
  va_end(targs);

  rc = len + 4 + 2; // 4 for length itself and 2 for message type
  write_exactly("vlog_sock_locked", fd, (uint32_t *) &rc, sizeof(uint32_t));
  write_exactly("vlog_sock_locked", fd, &log_err_type, sizeof(uint16_t));

  va_copy(targs, args);
  rc = vdprintf(fd, fmt, targs);
  if (rc < 0) die(1, NULL, "Couldn't send log message with vdprintf", "");
  va_end(targs);

  if (rc < len) { // we didn't write the whole message :-(
    rc = len - rc;
    fill = (char *) calloc(rc, 1);
    if (fill == NULL) die(1, NULL, "vlog_sock_locked fill", "");
    write_exactly("vlog_sock_locked fill", fd, fill, rc);
  }
}

void log_sock_locked(int fd, const char * fmt, ...) {
  va_list args;

  va_start(args, fmt);

  vlog_sock_locked(fd, fmt, args);

  va_end(args);
}

void die
(int exit_code, parameters * params, const char * parg, const char * fmt, ...)
{
  va_list argp, targs;
  int in_errno = errno;
  int fd = 0;

  if (params != NULL) {
    fd = params->ctl_sock;
    lock("die ctl_lock", &params->ctl_lock);
  }

  va_start(argp, fmt);
  va_copy(targs, argp);
  vsyslog(LOG_CRIT, fmt, targs);
  va_end(targs);

  if (fd != 0) vlog_sock_locked(fd, fmt, argp);
  va_end(argp);

  if (parg != NULL) {
    if (*parg != 0) {
      syslog(LOG_CRIT, "%s: %s", parg, strerror(in_errno));
      if (fd != 0) log_sock_locked(fd, "%s: %s", parg, strerror(in_errno));
    } else {
      syslog(LOG_CRIT, "%s", strerror(in_errno));
      if (fd != 0) log_sock_locked(fd, "%s", strerror(in_errno));
    }
  }

  if (fd != 0) close(fd); // flush
  exit(exit_code);
  // Nobody else should die before we terminate everything
  unlock("die ctl_lock", &params->ctl_lock);
}

void vlog_locked(parameters * params, const char * fmt, va_list args) {
  int rc;
  int fd = params->ctl_sock;
  va_list targs;

  if (fd != 0) vlog_sock_locked(fd, fmt, args);
  else {
    va_copy(targs, args);
    vsyslog(LOG_INFO, fmt, targs);
    va_end(targs);

    fd = params->logfile_fd;
    if (fd != 0) {
      va_copy(targs, args);
      rc = vdprintf(fd, fmt, targs);
      if (rc < 0) die(1, NULL, "Couldn't write log message with vdprintf", "");
      va_end(targs);
    }
  }
}

void vlog_time_locked(parameters * params, const char * fmt, va_list args) {
  int fd = params->logfile_fd;

  if (fd != 0 && params->ctl_sock == 0) log_timestamp(fd);
  vlog_locked(params, fmt, args);
}

void log_time_locked(parameters * params, const char * fmt, ...) {
  va_list args;

  va_start(args, fmt);

  vlog_time_locked(params, fmt, args);

  va_end(args);
}

void log_time(parameters * params, const char * fmt, ...) {
  va_list args;

  va_start(args, fmt);

  lock("log_time ctl_lock", &params->ctl_lock);
  vlog_time_locked(params, fmt, args);
  unlock("log_time ctl_lock", &params->ctl_lock);

  va_end(args);
}

typedef struct {
  parameters * params;
  char * msg;
} log_thread_state;

void * log_time_thread(void * log_state_ptr) {
  log_thread_state * log_state = log_state_ptr;

  log_time(log_state->params, log_state->msg);

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
  log_state->params = conn->params;

  va_start(args, fmt);
  if (vasprintf(&log_state->msg, fmt, args) == -1)
    die(1, conn->params, "Couldn't allocate thread_log_time message", "");
  va_end(args);

  // TODO: We currently spawn a new thread for every message. This is
  // far from ideal but fine for now as we anticipate thread-sensitive
  // log demand to be low.

  if ((errno = pthread_create(&logger, &detached, log_time_thread, log_state)))
    die(1, conn->params, "",
        "Couldn't create log thread for %s connection %s: ",
        conn->type_descr, conn->mount_point);
}

void log_continue_locked(parameters * params, const char * fmt, ...) {
  va_list args;

  va_start(args, fmt);

  vlog_locked(params, fmt, args);

  va_end(args);
}

void log_continue(parameters * params, const char * fmt, ...) {
  va_list args;

  va_start(args, fmt);

  lock("log_continue ctl_lock", &params->ctl_lock);
  vlog_locked(params, fmt, args);
  unlock("log_continue ctl_lock", &params->ctl_lock);

  va_end(args);
}
