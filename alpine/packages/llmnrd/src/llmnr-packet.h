/*
 * LLMNR (RFC 4705) packet format definitions
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

#ifndef LLMNR_PACKET_H
#define LLMNR_PACKET_H

#include <stdint.h>

#include "compiler.h"

#define LLMNR_IPV4_MCAST_ADDR	"224.0.0.252"
#define LLMNR_IPV6_MCAST_ADDR	"ff02:0:0:0:0:0:1:3"

#define LLMNR_UDP_PORT		5355

/*
 * LLMNR packet header (RFC 4795, section 2.1.1)
 */
struct llmnr_hdr {
	uint16_t id;
	uint16_t flags;
#define LLMNR_F_QR	0x8000
#define LLMNR_F_OPCODE	0x7800
#define LLMNR_F_C	0x0400
#define LLMNR_F_TC	0x0200
#define LLMNR_F_T	0x0100
#define LLMNR_F_RCODE	0x000f
	uint16_t qdcount;
	uint16_t ancount;
	uint16_t nscount;
	uint16_t arcount;
} __packed;

/* Maximum label length according to RFC 1035 */
#define LLMNR_LABEL_MAX_SIZE	63

/* TYPE values according to RFC1035, section 3.2.2 */
#define LLMNR_TYPE_A		1
#define LLMNR_TYPE_NS		2
#define LLMNR_TYPE_CNAME	5
#define LLMNR_TYPE_SOA		6
#define LLMNR_TYPE_PTR		12
#define LLMNR_TYPE_HINFO	13
#define LLMNR_TYPE_MINFO	14
#define LLMNR_TYPE_MX		15
#define LLMNR_TYPE_TXT		16
#define LLMNR_TYPE_AAAA		28	/* RFC 3596 */

/* QTYPE values according to RFC1035, section 3.2.3 */
#define LLMNR_QTYPE_A		LLMNR_TYPE_A
#define LLMNR_QTYPE_AAAA	LLMNR_TYPE_AAAA
#define LLMNR_QTYPE_ANY		255

/* CLASS values */
#define LLMNR_CLASS_IN		1

/* QCLASS values */
#define LLMNR_QCLASS_IN		LLMNR_CLASS_IN

/* Default RR TTL in seconds (RFC 4795, section 2.8) */
#define LLMNR_TTL_DEFAULT	30

#endif /* LLMNR_PACKET_H */
