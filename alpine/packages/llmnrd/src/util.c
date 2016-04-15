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

#include <stdlib.h>
#include <string.h>

#include "util.h"

void *xmalloc(size_t size)
{
	void *ptr;

	if (size == 0)
		panic("malloc: size 0\n");

	ptr = malloc(size);
	if (!ptr)
		panic("malloc: out of memory\n");

	return ptr;
}

void *xzalloc(size_t size)
{
	void *ptr = xmalloc(size);
	memset(ptr, 0, size);
	return ptr;
}

void *xrealloc(void *ptr, size_t size)
{
	void *newptr;

	if (size == 0)
		panic("realloc: size 0\n");

	newptr = realloc(ptr, size);
	if (!newptr) {
		free(ptr);
		panic("realloc: out of memory\n");
	}

	return newptr;
}

char *xstrdup(const char *s)
{
	size_t len = strlen(s) + 1;
	char *ret = xmalloc(len);

	memcpy(ret, s, len);

	return ret;
}
