#ifndef _TRANSFUSED_LOG_H_
#define _TRANSFUSED_LOG_H_

#include <stdarg.h>
#include <inttypes.h>

#include "transfused.h"

void die(int exit_code, parameters_t *params, const char *perror_arg,
	 const char *fmt, ...);

void vlog_locked(parameters_t *params, uint16_t msg_type,
		 const char *fmt, va_list args);
void vlog_time_locked(parameters_t *params, uint16_t msg_type,
		      const char *fmt, va_list args);

void log_time_locked(parameters_t *params, uint16_t msg_type,
		     const char *fmt, ...);

void log_time(parameters_t *params, const char *fmt, ...);
void log_notice_time(parameters_t *params, const char *fmt, ...);
void thread_log_time(connection_t *conn, const char *fmt, ...);
void log_continue_locked(parameters_t *params, const char *fmt, ...);
void log_continue(parameters_t *params, const char *fmt, ...);

#endif /* _TRANSFUSED_LOG_H_ */
