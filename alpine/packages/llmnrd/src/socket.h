/*
 * Copyright (C) 2014-2015 Tobias Klauser <tklauser@distanz.ch>
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

#ifndef SOCKET_H
#define SOCKET_H

#include <stdbool.h>
#include <stdint.h>

int socket_open_ipv4(uint16_t port);
int socket_open_ipv6(uint16_t port);
int socket_open_rtnl(void);

int socket_mcast_group_ipv4(int sock, unsigned int ifindex, bool join);
int socket_mcast_group_ipv6(int sock, unsigned int ifindex, bool join);

#endif /* SOCKET_H */
