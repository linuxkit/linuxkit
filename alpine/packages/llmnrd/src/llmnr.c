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

#define _GNU_SOURCE
#include <ctype.h>
#include <errno.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#include <arpa/inet.h>
#include <net/if.h>
#include <netinet/in.h>
#include <sys/select.h>

#include "iface.h"
#include "log.h"
#include "pkt.h"
#include "socket.h"

#include "iface.h"
#include "llmnr-packet.h"
#include "llmnr.h"

static int llmnr_sock_ipv4 = -1;
static int llmnr_sock_ipv6 = -1;
static bool llmnr_running = true;
/*
 * Host name in DNS name format (length octet + name + 0 byte)
 */
static char llmnr_hostname[LLMNR_LABEL_MAX_SIZE + 2];

static void llmnr_iface_event_handle(enum iface_event_type type, unsigned char af,
				     unsigned int ifindex)
{
	switch (af) {
	case AF_INET:
		socket_mcast_group_ipv4(llmnr_sock_ipv4, ifindex, type == IFACE_ADD);
		break;
	case AF_INET6:
		socket_mcast_group_ipv6(llmnr_sock_ipv6, ifindex, type == IFACE_ADD);
		break;
	default:
		/* ignore */
		break;
	}
}

int llmnr_init(const char *hostname, uint16_t port, bool ipv6)
{
	llmnr_hostname[0] = strlen(hostname);
	strncpy(&llmnr_hostname[1], hostname, LLMNR_LABEL_MAX_SIZE);
	llmnr_hostname[LLMNR_LABEL_MAX_SIZE + 1] = '\0';
	log_info("Starting llmnrd on port %u, hostname %s\n", port, hostname);

	llmnr_sock_ipv4 = socket_open_ipv4(port);
	if (llmnr_sock_ipv4 < 0)
		return -1;

	if (ipv6) {
		llmnr_sock_ipv6 = socket_open_ipv6(port);
		if (llmnr_sock_ipv6 < 0)
			return -1;
	}

	iface_register_event_handler(&llmnr_iface_event_handle);

	return 0;
}

static bool llmnr_name_matches(const uint8_t *query)
{
	uint8_t i, n = llmnr_hostname[0];

	/* length */
	if (query[0] != n)
		return false;
	/* NULL byte */
	if (query[1 + n] != 0)
		return false;

	for (i = 1; i < llmnr_hostname[0]; i++)
		if (tolower(query[i]) != tolower(llmnr_hostname[i]))
			return false;
	return true;
}

static void llmnr_respond(unsigned int ifindex, const struct llmnr_hdr *hdr,
			  const uint8_t *query, size_t query_len, int sock,
			  const struct sockaddr_storage *sst)
{
	uint16_t qtype, qclass;
	uint8_t name_len = query[0];
	/* skip name length & additional '\0' byte */
	const uint8_t *query_name_end = query + name_len + 2;
	size_t i, n, response_len;
	unsigned char family = AF_UNSPEC;
	/*
	 * arbitrary restriction to 16 addresses per interface for the
	 * sake of a simple, atomic interface
	 */
	struct sockaddr_storage addrs[16];
	struct pkt *p;
	struct llmnr_hdr *r;

	/* 4 bytes expected for QTYPE and QCLASS */
	if ((query_len - name_len - 2) < (sizeof(qtype) + sizeof(qclass)))
		return;

	qtype = ntohs(*((uint16_t *)query_name_end));
	qclass = ntohs(*((uint16_t *)query_name_end + 1));

	/* Ony IN queries supported */
	if (qclass != LLMNR_QCLASS_IN)
		return;

	switch (qtype) {
	case LLMNR_QTYPE_A:
		family = AF_INET;
		break;
	case LLMNR_QTYPE_AAAA:
		family = AF_INET6;
		break;
	case LLMNR_QTYPE_ANY:
		family = AF_UNSPEC;
		break;
	default:
		return;
	}

	n = iface_addr_lookup(ifindex, family, addrs, ARRAY_SIZE(addrs));

	/*
	 * This is the max response length (i.e. using all IPv6 addresses and
	 * not message compression). We might not use all of it.
	 */
	response_len = n * (1 + name_len + 1 + 2 + 2 + 4 + 2 + sizeof(struct in6_addr));
	p = pkt_alloc(sizeof(*hdr) + query_len + response_len);

	/* fill the LLMNR header */
	r = (struct llmnr_hdr *)pkt_put(p, sizeof(*r));
	r->id = hdr->id;
	/* response flag */
	r->flags = htons(LLMNR_F_QR);
	r->qdcount = hdr->qdcount;
	r->ancount = htons(n);
	r->nscount = 0;
	r->arcount = 0;

	/* copy the original question */
	memcpy(pkt_put(p, query_len), query, query_len);

	/* append an RR for each address */
	for (i = 0; i < n; i++) {
		void *addr;
		size_t addr_size;
		uint16_t type;

		if (addrs[i].ss_family == AF_INET) {
			struct sockaddr_in *sin = (struct sockaddr_in *)&addrs[i];
			addr = &sin->sin_addr;
			addr_size = sizeof(sin->sin_addr);
			type = LLMNR_TYPE_A;
		} else if (addrs[i].ss_family == AF_INET6) {
			struct sockaddr_in6 *sin6 = (struct sockaddr_in6 *)&addrs[i];
			addr = &sin6->sin6_addr;
			addr_size = sizeof(sin6->sin6_addr);
			type = LLMNR_TYPE_AAAA;
		} else
			continue;

		/* NAME */
		if (i == 0)
			memcpy(pkt_put(p, llmnr_hostname[0] + 2), llmnr_hostname, llmnr_hostname[0] + 2);
		else {
			/* message compression (RFC 1035, section 4.1.3) */
			uint16_t ptr = 0xC000 | (sizeof(*hdr) + query_len);
			pkt_put_u16(p, ntohs(ptr));
		}
		/* TYPE */
		pkt_put_u16(p, htons(type));
		/* CLASS */
		pkt_put_u16(p, htons(LLMNR_CLASS_IN));
		/* TTL */
		pkt_put_u32(p, htonl(LLMNR_TTL_DEFAULT));
		/* RDLENGTH */
		pkt_put_u16(p, htons(addr_size));
		/* RDATA */
		memcpy(pkt_put(p, addr_size), addr, addr_size);
	}

	if (sendto(sock, p->data, pkt_len(p), 0, (struct sockaddr *)sst, sizeof(*sst)) < 0)
		log_err("Failed to send response: %s\n", strerror(errno));

	pkt_free(p);
}

static void llmnr_packet_process(unsigned int ifindex, const uint8_t *pktbuf, size_t len,
				 int sock, const struct sockaddr_storage *sst)
{
	const struct llmnr_hdr *hdr = (const struct llmnr_hdr *)pktbuf;
	uint16_t flags, qdcount;
	const uint8_t *query;
	size_t query_len;
	uint8_t name_len;

	/* Query too short? */
	if (len < sizeof(struct llmnr_hdr))
		return;

	flags = ntohs(hdr->flags);
	qdcount = ntohs(hdr->qdcount);

	/* Query invalid as per RFC 4795, section 2.1.1? */
	if (((flags & (LLMNR_F_QR | LLMNR_F_OPCODE)) != 0) ||
	    qdcount != 1 || hdr->ancount != 0 || hdr->nscount != 0)
		return;

	query = pktbuf + sizeof(struct llmnr_hdr);
	query_len = len - sizeof(struct llmnr_hdr);
	name_len = query[0];
	/* Invalid name in query? */
	if (name_len == 0 || name_len >= query_len || query[1 + name_len] != 0)
		return;

	/* Authoritative? */
	if (llmnr_name_matches(query))
		llmnr_respond(ifindex, hdr, query, query_len, sock, sst);
}

static void llmnr_recv(int sock)
{
	uint8_t pktbuf[2048], aux[128];
	struct msghdr msg;
	struct iovec io;
	struct sockaddr_storage sin_r;
	struct cmsghdr *cmsg;
	ssize_t recvlen;
	int ifindex = -1;

	io.iov_base = pktbuf;
	io.iov_len = sizeof(pktbuf);

	memset(&msg, 0, sizeof(msg));
	msg.msg_name = &sin_r;
	msg.msg_namelen = sizeof(sin_r);
	msg.msg_iov = &io;
	msg.msg_iovlen = 1;
	msg.msg_control = aux;
	msg.msg_controllen = sizeof(aux);

	if ((recvlen = recvmsg(sock, &msg, 0)) < 0) {
		if (errno != EINTR)
			log_err("Failed to receive packet: %s\n", strerror(errno));
		return;
	}

	for (cmsg = CMSG_FIRSTHDR(&msg); cmsg; cmsg = CMSG_NXTHDR(&msg, cmsg)) {
		if (cmsg->cmsg_level == IPPROTO_IP && cmsg->cmsg_type == IP_PKTINFO) {
			struct in_pktinfo *in = (struct in_pktinfo *)CMSG_DATA(cmsg);
			ifindex = in->ipi_ifindex;
		} else if (cmsg->cmsg_level == IPPROTO_IPV6 && cmsg->cmsg_type == IPV6_PKTINFO) {
			struct in6_pktinfo *in6 = (struct in6_pktinfo *)CMSG_DATA(cmsg);
			ifindex = in6->ipi6_ifindex;
		}
	}

	if (ifindex >= 0)
		llmnr_packet_process(ifindex, pktbuf, recvlen, sock,
				     (const struct sockaddr_storage *)&sin_r);
	else
		log_warn("Could not get interface of incoming packet\n");
}

int llmnr_run(void)
{
	int ret = -1;

	while (llmnr_running) {
		fd_set rfds;
		struct timeval tv;
		int nfds, ret;

		FD_ZERO(&rfds);
		FD_SET(llmnr_sock_ipv4, &rfds);
		if (llmnr_sock_ipv6 >= 0) {
			FD_SET(llmnr_sock_ipv6, &rfds);
			nfds = max(llmnr_sock_ipv4, llmnr_sock_ipv6) + 1;
		} else
			nfds = llmnr_sock_ipv4 + 1;

		tv.tv_sec = 0;
		tv.tv_usec = 200;

		ret = select(nfds, &rfds, NULL, NULL, &tv);
		if (ret < 0) {
			if (errno != EINTR)
				log_err("Failed to select() on socket: %s\n", strerror(errno));
			goto out;
		} else if (ret) {
			if (FD_ISSET(llmnr_sock_ipv4, &rfds))
				llmnr_recv(llmnr_sock_ipv4);
			if (llmnr_sock_ipv6 >= 0 && FD_ISSET(llmnr_sock_ipv6, &rfds))
				llmnr_recv(llmnr_sock_ipv6);
		}
	}

	ret = 0;
out:
	close(llmnr_sock_ipv4);
	if (llmnr_sock_ipv6 >= 0)
		close(llmnr_sock_ipv6);
	return ret;
}

void llmnr_stop(void)
{
	llmnr_running = false;
}
