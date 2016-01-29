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

#include <errno.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#include <arpa/inet.h>
#include <linux/netlink.h>
#include <linux/rtnetlink.h>
#include <net/if.h>
#include <netinet/in.h>
#include <sys/socket.h>
#include <sys/types.h>

#include "llmnr-packet.h"
#include "log.h"
#include "socket.h"

static const int YES = 1;

int socket_open_ipv4(uint16_t port)
{
	int sock;
	struct sockaddr_in sa;

	sock = socket(AF_INET, SOCK_DGRAM, 0);
	if (sock < 0) {
		log_err("Failed to open UDP socket: %s\n", strerror(errno));
		return -1;
	}

	/* pass pktinfo struct on received packets */
	if (setsockopt(sock, IPPROTO_IP, IP_PKTINFO, &YES, sizeof(YES)) < 0) {
		log_err("Failed to set IPv4 packet info socket option: %s\n", strerror(errno));
		goto err;
	}

	/* bind the socket */
	memset(&sa, 0, sizeof(sa));
	sa.sin_family = AF_INET;
	sa.sin_addr.s_addr = INADDR_ANY;
	sa.sin_port = htons(port);

	if (bind(sock, (struct sockaddr *)&sa, sizeof(sa)) < 0) {
		log_err("Failed to bind() socket: %s\n", strerror(errno));
		goto err;
	}

	return sock;
err:
	close(sock);
	return -1;
}

int socket_open_ipv6(uint16_t port)
{
	int sock, opt_pktinfo;
	struct sockaddr_in6 sa;

	sock = socket(AF_INET6, SOCK_DGRAM, 0);
	if (sock < 0) {
		log_err("Failed to open UDP socket: %s\n", strerror(errno));
		return -1;
	}

	/* pass pktinfo struct on received packets */
#if defined(IPV6_RECVPKTINFO)
	opt_pktinfo = IPV6_RECVPKTINFO;
#elif defined(IPV6_PKTINFO)
	opt_pktinfo = IPV6_PKTINFO;
#endif
	if (setsockopt(sock, IPPROTO_IPV6, opt_pktinfo, &YES, sizeof(YES)) < 0) {
		log_err("Failed to set IPv6 packet info socket option: %s\n", strerror(errno));
		goto err;
	}

	/* IPv6 only socket */
	if (setsockopt(sock, IPPROTO_IPV6, IPV6_V6ONLY, &YES, sizeof(YES)) < 0) {
		log_err("Failed to set IPv6 only socket option: %s\n", strerror(errno));
		goto err;
	}

	/* bind the socket */
	memset(&sa, 0, sizeof(sa));
	sa.sin6_family = AF_INET6;
	sa.sin6_port = htons(port);

	if (bind(sock, (struct sockaddr *)&sa, sizeof(sa)) < 0) {
		log_err("Failed to bind() socket: %s\n", strerror(errno));
		goto err;
	}

	return sock;
err:
	close(sock);
	return -1;
}

int socket_open_rtnl(void)
{
	int sock;
	struct sockaddr_nl sa;

	sock = socket(AF_NETLINK, SOCK_RAW, NETLINK_ROUTE);
	if (sock < 0) {
		log_err("Failed to open netlink route socket: %s\n", strerror(errno));
		return -1;
	}

	memset(&sa, 0, sizeof(sa));
	sa.nl_family = AF_NETLINK;
	/*
	 * listen for following events:
	 * - network interface create/delete/up/down
	 * - IPv4 address add/delete
	 * - IPv6 address add/delete
	 */
	sa.nl_groups = RTMGRP_LINK | RTMGRP_IPV4_IFADDR | RTMGRP_IPV6_IFADDR;

	if (bind(sock, (struct sockaddr *)&sa, sizeof(sa)) < 0) {
		log_err("Failed to bind() netlink socket: %s\n", strerror(errno));
		goto err;
	}

	return sock;
err:
	close(sock);
	return -1;
}

int socket_mcast_group_ipv4(int sock, unsigned int ifindex, bool join)
{
	struct ip_mreqn mreq;
	char ifname[IF_NAMESIZE];

	/* silently ignore, we might not be listening on an IPv4 socket */
	if (sock < 0)
		return -1;

	memset(&mreq, 0, sizeof(mreq));
	mreq.imr_ifindex = ifindex;
	mreq.imr_address.s_addr = INADDR_ANY;
	inet_pton(AF_INET, LLMNR_IPV4_MCAST_ADDR, &mreq.imr_multiaddr);

	if (setsockopt(sock, IPPROTO_IP, join ? IP_ADD_MEMBERSHIP : IP_DROP_MEMBERSHIP,
		       &mreq, sizeof(mreq)) < 0) {
		log_err("Failed to join IPv4 multicast group on interface %s: %s\n",
			if_indextoname(ifindex, ifname), strerror(errno));
		return -1;
	}

	return 0;
}

int socket_mcast_group_ipv6(int sock, unsigned int ifindex, bool join)
{
	struct ipv6_mreq mreq6;
	char ifname[IF_NAMESIZE];

	/* silently ignore, we might not be listening on an IPv6 socket */
	if (sock < 0)
		return -1;

	memset(&mreq6, 0, sizeof(mreq6));
	mreq6.ipv6mr_interface = ifindex;
	inet_pton(AF_INET6, LLMNR_IPV6_MCAST_ADDR, &mreq6.ipv6mr_multiaddr);

	if (setsockopt(sock, IPPROTO_IPV6, join ? IPV6_ADD_MEMBERSHIP : IPV6_DROP_MEMBERSHIP,
		       &mreq6, sizeof(mreq6)) < 0) {
		log_err("Failed to join IPv6 multicast group on interface %s: %s\n",
			if_indextoname(ifindex, ifname), strerror(errno));
		return -1;
	}

	return 0;
}
