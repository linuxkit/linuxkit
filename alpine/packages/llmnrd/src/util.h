/*
 * Copyright (C) 2014-2015 Tobias Klauser <tklauser@distanz.ch>
 * Copyright (C) 2009-2012 Daniel Borkmann
 *
 * This file is part of llmnrd.
 *
 * llmnrd is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, version 2 of the License.
 *
 * llmnrd is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with llmnrd.  If not, see <http://www.gnu.org/licenses/>.
 */

#ifndef UTIL_H
#define UTIL_H

#include <stdarg.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>

#include "compiler.h"

#define ARRAY_SIZE(x)	(sizeof(x) / sizeof((x)[0]))

/*
 * min()/max() macros with strict type-checking.
 * Taken from linux/kernel.h
 */
#undef min
#define min(x, y) ({			\
	typeof(x) _min1 = (x);		\
	typeof(y) _min2 = (y);		\
	(void) (&_min1 == &_min2);	\
	_min1 < _min2 ? _min1 : _min2; })

#undef max
#define max(x, y) ({			\
	typeof(x) _max1 = (x);		\
	typeof(y) _max2 = (y);		\
	(void) (&_max1 == &_max2);	\
	_max1 > _max2 ? _max1 : _max2; })

static inline void __noreturn panic(const char *fmt, ...)
{
	va_list vl;

	va_start(vl, fmt);
	vfprintf(stderr, fmt, vl);
	va_end(vl);

	exit(EXIT_FAILURE);
}

void *xmalloc(size_t size) __warn_unused_result;
void *xzalloc(size_t size) __warn_unused_result;
void *xrealloc(void *ptr, size_t size) __warn_unused_result;
char *xstrdup(const char *s) __warn_unused_result;

static inline bool xstreq(const char *str1, const char *str2)
{
	size_t n = strlen(str1);

	if (n != strlen(str2))
		return false;
	if (strncmp(str1, str2, n) != 0)
		return false;

	return true;
}

#endif /* UTIL_H */
