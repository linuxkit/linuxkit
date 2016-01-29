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

#include <assert.h>
#include <errno.h>
#include <pthread.h>
#include <stdint.h>
#include <string.h>
#include <unistd.h>

#include <arpa/inet.h>
#include <linux/netlink.h>
#include <linux/rtnetlink.h>
#include <net/if.h>
#include <netinet/in.h>
#include <sys/socket.h>
#include <sys/types.h>

#include "err.h"
#include "list.h"
#include "log.h"
#include "socket.h"
#include "util.h"

#include "iface.h"

static bool iface_running = true;
static pthread_t iface_thread;
static iface_event_handler_t iface_event_handler;

struct iface_record {
	struct list_head list;
	unsigned int index;
	struct sockaddr_storage *addrs;
	size_t size;
};

static struct list_head iface_list_head;
static pthread_mutex_t iface_list_mutex;

size_t iface_addr_lookup(unsigned int ifindex, unsigned char family,
			 struct sockaddr_storage *addrs, size_t addrs_size)
{
	struct iface_record *rec;
	size_t n = 0;

	if (!addrs)
		return 0;

	pthread_mutex_lock(&iface_list_mutex);

	list_for_each_entry(rec, &iface_list_head, list) {
		if (rec->index == ifindex) {
			size_t i;

			for (i = 0; i < rec->size && n < addrs_size; i++) {
				if (family == AF_UNSPEC || family == rec->addrs[i].ss_family) {
					memcpy(&addrs[n], &rec->addrs[i], sizeof(addrs[n]));
					n++;
				}
			}
			break;
		}
	}

	pthread_mutex_unlock(&iface_list_mutex);

	return n;
}

static bool iface_record_addr_eq(const struct sockaddr_storage *addr1,
				 const struct sockaddr_storage *addr2)
{
	int family = addr1->ss_family;

	if (family != addr2->ss_family)
		return false;

	if (family == AF_INET) {
		const struct sockaddr_in *sin1 = (const struct sockaddr_in *)addr1;
		const struct sockaddr_in *sin2 = (const struct sockaddr_in *)addr2;

		return memcmp(&sin1->sin_addr, &sin2->sin_addr, sizeof(sin1->sin_addr)) == 0;
	} else if (family == AF_INET6) {
		const struct sockaddr_in6 *sin1 = (const struct sockaddr_in6 *)addr1;
		const struct sockaddr_in6 *sin2 = (const struct sockaddr_in6 *)addr2;

		return memcmp(&sin1->sin6_addr, &sin2->sin6_addr, sizeof(sin1->sin6_addr)) == 0;
	} else {
		/* This should never happen */
		log_warn("Unsupported address family: %d\n", family);
		return memcmp(addr1, addr2, sizeof(*addr1));
	}
}

static void iface_record_addr_add(struct iface_record *rec, struct sockaddr_storage *addr)
{
	size_t i;
	struct sockaddr_storage *addrs = rec->addrs;

	for (i = 0; i < rec->size; i++) {
		/* Address already in record? */
		if (iface_record_addr_eq(&addrs[i], addr))
			return;
	}

	addrs = xrealloc(rec->addrs, (rec->size + 1) * sizeof(*addr));
	memcpy(&addrs[rec->size], addr, sizeof(*addr));
	rec->addrs = addrs;
	rec->size++;
}

static void iface_record_addr_del(struct iface_record *rec, struct sockaddr_storage *addr)
{
	if (rec->size > 1) {
		size_t i, j = 0;
		struct sockaddr_storage *addrs = xmalloc((rec->size - 1) * sizeof(*addr));

		for (i = 0; i < rec->size; i++) {
			if (!iface_record_addr_eq(&rec->addrs[i], addr)) {
				memcpy(&addrs[j], &rec->addrs[i], sizeof(addrs[j]));
				j++;
			}
		}

		if (j == i - 1) {
			free(rec->addrs);
			rec->addrs = addrs;
			rec->size--;
		} else {
			char as[INET6_ADDRSTRLEN];
			inet_ntop(addr->ss_family, addr + sizeof(addr->ss_family), as, sizeof(as));
			log_err("Address %s to delete not found in records\n", as);
		}
	} else if (rec->size == 1) {
		free(rec->addrs);
		rec->addrs = NULL;
		rec->size = 0;
	}
}

static inline void fill_sockaddr_storage(struct sockaddr_storage *sst,
					 unsigned char family, const void *addr)
{
	sst->ss_family = family;
	if (family == AF_INET) {
		struct sockaddr_in *sin = (struct sockaddr_in *)sst;
		memcpy(&sin->sin_addr, addr, sizeof(sin->sin_addr));
	} else if (family == AF_INET6) {
		struct sockaddr_in6 *sin6 = (struct sockaddr_in6 *)sst;
		memcpy(&sin6->sin6_addr, addr, sizeof(sin6->sin6_addr));
	}
}

static void iface_addr_add(unsigned int index, unsigned char family, const void *addr)
{
	struct iface_record *rec;
	struct sockaddr_storage sst;

	fill_sockaddr_storage(&sst, family, addr);

	pthread_mutex_lock(&iface_list_mutex);

	list_for_each_entry(rec, &iface_list_head, list)
		if (rec->index == index)
			goto add;

	rec = xzalloc(sizeof(*rec));
	INIT_LIST_HEAD(&rec->list);
	rec->index = index;

	list_add_tail(&rec->list, &iface_list_head);
add:
	iface_record_addr_add(rec, &sst);
	pthread_mutex_unlock(&iface_list_mutex);
}

static void iface_addr_del(unsigned int index, unsigned char family, const void *addr)
{
	struct iface_record *rec;
	struct sockaddr_storage sst;

	fill_sockaddr_storage(&sst, family, addr);

	pthread_mutex_lock(&iface_list_mutex);

	list_for_each_entry(rec, &iface_list_head, list) {
		if (rec->index == index) {
			iface_record_addr_del(rec, &sst);
			break;
		}
	}

	pthread_mutex_unlock(&iface_list_mutex);
}

static void iface_nlmsg_change_link(const struct nlmsghdr *nlh __unused)
{
	/* TODO */
}

static void iface_nlmsg_change_addr(const struct nlmsghdr *nlh)
{
	struct ifaddrmsg *ifa = NLMSG_DATA(nlh);
	struct rtattr *rta;
	size_t rtalen = nlh->nlmsg_len - NLMSG_SPACE(sizeof(*ifa));
	unsigned char family = ifa->ifa_family;
	unsigned int index = ifa->ifa_index;
	char ifname[IF_NAMESIZE];

	/* don't report temporary addresses */
	if ((ifa->ifa_flags & (IFA_F_TEMPORARY | IFA_F_TENTATIVE)) != 0)
		return;

	if_indextoname(index, ifname);

	rta = (struct rtattr *)((const uint8_t *)nlh + NLMSG_SPACE(sizeof(*ifa)));
	for ( ; RTA_OK(rta, rtalen); rta = RTA_NEXT(rta, rtalen)) {
		char addr[INET6_ADDRSTRLEN];
		enum iface_event_type type;

		if (rta->rta_type != IFA_ADDRESS)
			continue;

		if (!inet_ntop(family, RTA_DATA(rta), addr, sizeof(addr)))
			strncpy(addr, "<unknown>", sizeof(addr) - 1);

		if (nlh->nlmsg_type == RTM_NEWADDR) {
			iface_addr_add(index, family, RTA_DATA(rta));
			type = IFACE_ADD;
		} else if (nlh->nlmsg_type == RTM_DELADDR) {
			iface_addr_del(index, family, RTA_DATA(rta));
			type = IFACE_DEL;
		} else {
			/* This case shouldn't occur */
			continue;
		}

		if (iface_event_handler)
			(*iface_event_handler)(type, family, index);

		log_info("%s IPv%c address %s on interface %s\n",
			 type == IFACE_ADD ? "Added" : "Deleted",
			 family == AF_INET ? '4' : '6', addr, ifname);
	}
}

static int iface_nlmsg_process(const struct nlmsghdr *nlh, size_t len)
{
	for ( ; len > 0; nlh = NLMSG_NEXT(nlh, len)) {
		struct nlmsgerr *err;

		if (!NLMSG_OK(nlh, len)) {
			log_err("netlink message truncated\n");
			return -1;
		}

		switch (nlh->nlmsg_type) {
		case RTM_NEWADDR:
		case RTM_DELADDR:
			iface_nlmsg_change_addr(nlh);
			break;
		case RTM_NEWLINK:
		case RTM_DELLINK:
			iface_nlmsg_change_link(nlh);
			break;
		case NLMSG_ERROR:
			err = NLMSG_DATA(nlh);
			log_err("netlink error: %s\n", strerror(-(err->error)));
			break;
		case NLMSG_DONE:
			if (!NLMSG_OK(nlh, len)) {
				log_err("netlink message truncated\n");
				return -1;
			} else
				return 0;
		default:
			/* log_warn("Unknown netlink message type: 0x%x\n", nlh->nlmsg_type); */
			break;
		}
	}

	return 0;
}

static int iface_rtnl_enumerate(int sock, uint16_t type, unsigned char family)
{
	struct {
		struct nlmsghdr n;
		struct rtgenmsg r;
	} req;
	ssize_t recvlen;
	uint8_t pktbuf[8192];

	memset(&req, 0, sizeof(req));
	req.n.nlmsg_len = NLMSG_LENGTH(sizeof(req.r));
	req.n.nlmsg_flags = NLM_F_REQUEST | NLM_F_DUMP;
	req.n.nlmsg_type = type;
	req.r.rtgen_family = family;

	if (send(sock, &req, req.n.nlmsg_len, 0) < 0) {
		log_err("Failed to send netlink enumeration message: %s\n", strerror(errno));
		return -1;
	}

	if ((recvlen = recv(sock, pktbuf, sizeof(pktbuf), 0)) < 0) {
		if (errno != EINTR)
			log_err("Failed to receive netlink message: %s\n", strerror(errno));
		return -1;
	}

	return iface_nlmsg_process((const struct nlmsghdr *)pktbuf, recvlen);
}

void iface_register_event_handler(iface_event_handler_t event_handler)
{
	iface_event_handler = event_handler;
}

int iface_run(void)
{
	int ret = -1;
	int sock;

	INIT_LIST_HEAD(&iface_list_head);
	if (pthread_mutex_init(&iface_list_mutex, NULL) != 0) {
		log_err("Failed to initialize interface list mutex\n");
		return -1;
	}

	sock = socket_open_rtnl();
	if (sock < 0)
		return -1;

	/* send RTM_GETADDR request to initially populate the interface list */
	if (iface_rtnl_enumerate(sock, RTM_GETADDR, AF_INET) < 0)
		return -1;
	if (iface_rtnl_enumerate(sock, RTM_GETADDR, AF_INET6) < 0)
		return -1;

	while (iface_running) {
		ssize_t recvlen;
		uint8_t pktbuf[8192];

		if ((recvlen = recv(sock, pktbuf, sizeof(pktbuf), 0)) < 0) {
			if (errno != EINTR)
				log_err("Failed to receive netlink message: %s\n", strerror(errno));
			goto out;
		}

		if (iface_nlmsg_process((const struct nlmsghdr *)pktbuf, recvlen) < 0)
			log_warn("Error processing netlink message\n");
	}

	pthread_mutex_destroy(&iface_list_mutex);
	ret = 0;
out:
	close(sock);
	return ret;
}

static void* iface_run_wrapper(void *data __unused)
{
	return ERR_PTR(iface_run());
}

int iface_start_thread(void)
{
	if (pthread_create(&iface_thread, NULL, iface_run_wrapper, NULL) < 0) {
		log_err("Failed to start interface monitoring thread\n");
		return -1;
	}

	return 0;
}

void iface_stop(void)
{
	iface_running = false;
}
