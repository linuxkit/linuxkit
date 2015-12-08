/*-
 * Copyright (c) 2012 NetApp, Inc.
 * Copyright (c) 2015 xhyve developers
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 * 1. Redistributions of source code must retain the above copyright
 *    notice, this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright
 *    notice, this list of conditions and the following disclaimer in the
 *    documentation and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY NETAPP, INC ``AS IS'' AND
 * ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED.  IN NO EVENT SHALL NETAPP, INC OR CONTRIBUTORS BE LIABLE
 * FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
 * DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
 * OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
 * HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
 * LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY
 * OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
 * SUCH DAMAGE.
 *
 * $FreeBSD$
 */

#include <stdint.h>
#include <sys/sysctl.h>
#include <xhyve/support/misc.h>
#include <xhyve/support/specialreg.h>
#include <xhyve/vmm/vmm.h>
#include <xhyve/vmm/vmm_host.h>

static struct xsave_limits vmm_xsave_limits;

void
vmm_host_state_init(void)
{
	uint32_t avx1_0, regs[4];
	size_t ln;

	vmm_xsave_limits.xsave_enabled = 0;

	ln = sizeof(uint32_t);
	if (!sysctlbyname("hw.optional.avx1_0", &avx1_0, &ln, NULL, 0) && avx1_0) {
		cpuid_count(0xd, 0x0, regs);
		vmm_xsave_limits.xsave_enabled = 1;
		vmm_xsave_limits.xcr0_allowed = XFEATURE_AVX;
		vmm_xsave_limits.xsave_max_size = regs[1];
	}
}

const struct xsave_limits *
vmm_get_xsave_limits(void)
{
	return (&vmm_xsave_limits);
}
