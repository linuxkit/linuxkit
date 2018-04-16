#include <asl.h>
#include <pwd.h>
#include <fcntl.h>
#include <stdio.h>
#include <time.h>

#include <SystemConfiguration/SystemConfiguration.h>

#include <xhyve/log.h>

static aslclient log_client = NULL;
static aslmsg log_msg = NULL;

static unsigned char buf[4096];
/* Index of the _next_ character to insert in the buffer. */
static size_t buf_idx = 0;

/* asl is deprecated in favor of os_log starting with macOS 10.12.  */
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"

/* Initialize ASL logger and local buffer. */
void log_init(void)
{
	log_client = asl_open(NULL, NULL, 0);
	log_msg = asl_new(ASL_TYPE_MSG);
}


/* Send the content of the buffer to the logger. */
static void log_flush(void)
{
	buf[buf_idx] = 0;
	asl_log(log_client, log_msg, ASL_LEVEL_NOTICE, "%s", buf);
	buf_idx = 0;
}


/* Send one character to the logger: wait for full lines before actually sending. */
void log_put(uint8_t c)
{
	if ((c == '\n') || (c == 0)) {
		log_flush();
	} else {
		if (buf_idx + 2 >= sizeof(buf)) {
			log_flush();
		}
		buf[buf_idx] = c;
		++buf_idx;
	}
}
