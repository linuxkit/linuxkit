#include <errno.h>
#include <stdio.h>
#include <string.h>

#include "log.h"

int verbose = 0;

void fatal(const char *msg)
{
	ERROR("%s Error: %d. %s", msg, errno, strerror(errno));
	exit(1);
}
