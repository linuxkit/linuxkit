# include <stdio.h>
# include <stdarg.h>
# include <stdlib.h>
# include <string.h>
static void error_at_line(int status, int errnum, const char *filename,
                          unsigned int linenum, const char *format, ...)
{
	va_list ap;

	fflush(stdout);

	if (filename != NULL)
		fprintf(stderr, "%s:%u: ", filename, linenum);

	va_start(ap, format);
	vfprintf(stderr, format, ap);
	va_end(ap);

	if (errnum != 0)
		fprintf(stderr, ": %s", strerror(errnum));

	fprintf(stderr, "\n");

	if (status != 0)
		exit(status);
}

#define error(status, errnum, format...) \
	error_at_line(status, errnum, NULL, 0, format)
