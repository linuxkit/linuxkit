/*
 * Simple doubly linked list, based on the Linux kernel linked list.
 *
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

#ifndef LIST_H
#define LIST_H

#include <stdbool.h>

#include "compiler.h"

struct list_head {
	struct list_head *next, *prev;
};

static inline void INIT_LIST_HEAD(struct list_head *list)
{
	list->next = list;
	list->prev = list;
}

static inline void __list_add(struct list_head *obj,
			      struct list_head *prev,
			      struct list_head *next)
{
	prev->next = obj;
	obj->prev = prev;
	obj->next = next;
	next->prev = obj;
}

static inline void list_add_tail(struct list_head *obj, struct list_head *head)
{
	__list_add(obj, head->prev, head);
}

static inline void list_add_head(struct list_head *obj, struct list_head *head)
{
	__list_add(obj, head, head->next);
}

static inline void list_del(struct list_head *obj)
{
	obj->next->prev = obj->prev;
	obj->prev->next = obj->next;
}

static inline bool list_empty(struct list_head *head)
{
	return head->next == head;
}

#define list_entry(ptr, type, member)	container_of(ptr, type, member)

#define list_for_each_entry(pos, head, member)				\
	for (pos = list_entry((head)->next, typeof(*pos), member);	\
	     &(pos)->member != (head);					\
	     (pos) = list_entry((pos)->member.next, typeof(*(pos)), member))

#define list_for_each_entry_safe(pos, n, head, member)				\
	for (pos = list_entry((head)->next, typeof(*pos), member),		\
		n = list_entry(pos->member.next, typeof(*pos), member);	\
	     &(pos)->member != (head);						\
	     pos = n, n = list_entry(n->member.next, typeof(*n), member))

#endif /* LIST_H */
