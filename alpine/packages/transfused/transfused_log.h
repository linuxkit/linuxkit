#include <stdarg.h>

#include "transfused.h"

void die
(int exit_code, parameters * params, const char * perror_arg,
 const char * fmt, ...);

void vlog_locked(parameters * params, const char * fmt, va_list args);

void vlog_time_locked(parameters * params, const char * fmt, va_list args);

void log_time_locked(parameters * params, const char * fmt, ...);

void log_time(parameters * params, const char * fmt, ...);

void thread_log_time(connection_t * conn, const char * fmt, ...);

void log_continue_locked(parameters * params, const char * fmt, ...);

void log_continue(parameters * params, const char * fmt, ...);
