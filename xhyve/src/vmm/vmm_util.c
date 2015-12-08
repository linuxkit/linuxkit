/*-
 * Copyright (c) 2011 NetApp, Inc.
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

#include <stdio.h>
#include <xhyve/support/misc.h>
#include <xhyve/vmm/vmm_util.h>

struct trapframe {
	register_t	tf_rdi;
	register_t	tf_rsi;
	register_t	tf_rdx;
	register_t	tf_rcx;
	register_t	tf_r8;
	register_t	tf_r9;
	register_t	tf_rax;
	register_t	tf_rbx;
	register_t	tf_rbp;
	register_t	tf_r10;
	register_t	tf_r11;
	register_t	tf_r12;
	register_t	tf_r13;
	register_t	tf_r14;
	register_t	tf_r15;
	uint32_t	tf_trapno;
	uint16_t	tf_fs;
	uint16_t	tf_gs;
	register_t	tf_addr;
	uint32_t	tf_flags;
	uint16_t	tf_es;
	uint16_t	tf_ds;
	/* below portion defined in hardware */
	register_t	tf_err;
	register_t	tf_rip;
	register_t	tf_cs;
	register_t	tf_rflags;
	register_t	tf_rsp;
	register_t	tf_ss;
};

#define	DUMP_REG(x)	printf(#x "\t\t0x%016lx\n", (long)(tf->tf_ ## x))
#define	DUMP_SEG(x)	printf(#x "\t\t0x%04x\n", (unsigned)(tf->tf_ ## x))
void
dump_trapframe(struct trapframe *tf)
{
	DUMP_REG(rdi);
	DUMP_REG(rsi);
	DUMP_REG(rdx);
	DUMP_REG(rcx);
	DUMP_REG(r8);
	DUMP_REG(r9);
	DUMP_REG(rax);
	DUMP_REG(rbx);
	DUMP_REG(rbp);
	DUMP_REG(r10);
	DUMP_REG(r11);
	DUMP_REG(r12);
	DUMP_REG(r13);
	DUMP_REG(r14);
	DUMP_REG(r15);
	DUMP_REG(trapno);
	DUMP_REG(addr);
	DUMP_REG(flags);
	DUMP_REG(err);
	DUMP_REG(rip);
	DUMP_REG(rflags);
	DUMP_REG(rsp);
	DUMP_SEG(cs);
	DUMP_SEG(ss);
	DUMP_SEG(fs);
	DUMP_SEG(gs);
	DUMP_SEG(es);
	DUMP_SEG(ds);
}
