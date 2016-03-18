#include "transfused.h"

void die(int exit_code, const char * perror_arg, const char * fmt, ...);

void vlog_locked(connection_t * conn, const char * fmt, va_list args);

void vlog_time_locked(connection_t * conn, const char * fmt, va_list args);

void log_time_locked(connection_t * connection, const char * fmt, ...);

void log_time(connection_t * connection, const char * fmt, ...);

void log_continue_locked(connection_t * connection, const char * fmt, ...);

void log_continue(connection_t * connection, const char * fmt, ...);
