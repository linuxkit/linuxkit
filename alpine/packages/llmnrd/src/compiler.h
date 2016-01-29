/*
 * Copyright (C) 2015 Tobias Klauser <tklauser@distanz.ch>
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

#ifndef COMPILER_H
#define COMPILER_H

#ifdef __GNUC__
# define __noreturn		__attribute__((noreturn))
# define __warn_unused_result	__attribute__((warn_unused_result))
# define __packed		__attribute__((packed))
# define __unused		__attribute__((unused))
# ifndef offsetof
#  define offsetof(a, b)	__builtin_offsetof(a, b)
# endif
#else
# define __noreturn
# define __packed
# define __unused
#endif

#ifndef offsetof
# define offsetof(type, member)	((size_t) &((type *)0)->member)
#endif

#ifndef container_of
# define container_of(ptr, type, member) ({			\
	const typeof(((type *)0)->member) *__mptr = (ptr);	\
	(type *)((char *)__mptr - offsetof(type, member));})
#endif

#endif /* COMPILER_H */
