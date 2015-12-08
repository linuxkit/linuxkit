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
#include <string.h>
#include <errno.h>
#include <assert.h>
#include <sys/uio.h>
#include <xhyve/support/misc.h>
#include <xhyve/support/linker_set.h>
#include <xhyve/support/psl.h>
#include <xhyve/support/segments.h>
#include <xhyve/vmm/vmm_api.h>
#include <xhyve/xhyve.h>
#include <xhyve/inout.h>

SET_DECLARE(inout_port_set, struct inout_port);

#define	MAX_IOPORTS	(1 << 16)

#define	VERIFY_IOPORT(port, size) \
	assert((port) >= 0 && (size) > 0 && ((port) + (size)) <= MAX_IOPORTS)

#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wpadded"
static struct {
	const char *name;
	int flags;
	inout_func_t handler;
	void *arg;
} inout_handlers[MAX_IOPORTS];
#pragma clang diagnostic pop

static int
default_inout(UNUSED int vcpu, int in, UNUSED int port, int bytes,
	uint32_t *eax, UNUSED void *arg)
{
	if (in) {
		switch (bytes) {
		case 4:
			*eax = 0xffffffff;
			break;
		case 2:
			*eax = 0xffff;
			break;
		case 1:
			*eax = 0xff;
			break;
		}
	}
	
	return (0);
}

static void 
register_default_iohandler(int start, int size)
{
	struct inout_port iop;
	
	VERIFY_IOPORT(start, size);

	bzero(&iop, sizeof(iop));
	iop.name = "default";
	iop.port = start;
	iop.size = size;
	iop.flags = (int) (IOPORT_F_INOUT | IOPORT_F_DEFAULT);
	iop.handler = default_inout;

	register_inout(&iop);
}

static int
update_register(int vcpuid, enum vm_reg_name reg,
	uint64_t val, int size)
{
	int error;
	uint64_t origval;

	switch (size) {
	case 1:
	case 2:
		error = xh_vm_get_register(vcpuid, reg, &origval);
		if (error)
			return (error);
		val &= vie_size2mask(size);
		val |= origval & ~vie_size2mask(size);
		break;
	case 4:
		val &= 0xffffffffUL;
		break;
	case 8:
		break;
	default:
		return (EINVAL);
	}

	return xh_vm_set_register(vcpuid, reg, val);
}

int
emulate_inout(int vcpu, struct vm_exit *vmexit, int strict)
{
	int addrsize, bytes, flags, in, port, prot, rep;
	uint32_t eax, val;
	inout_func_t handler;
	void *arg;
	int error, fault, retval;
	enum vm_reg_name idxreg;
	uint64_t gla, index, iterations, count;
	struct vm_inout_str *vis;
	struct iovec iov[2];

	bytes = vmexit->u.inout.bytes;
	in = vmexit->u.inout.in;
	port = vmexit->u.inout.port;

	assert(port < MAX_IOPORTS);
	assert(bytes == 1 || bytes == 2 || bytes == 4);

	handler = inout_handlers[port].handler;

	if (strict && handler == default_inout)
		return (-1);

	flags = inout_handlers[port].flags;
	arg = inout_handlers[port].arg;

	if (in) {
		if (!(flags & IOPORT_F_IN))
			return (-1);
	} else {
		if (!(flags & IOPORT_F_OUT))
			return (-1);
	}

	retval = 0;
	if (vmexit->u.inout.string) {
		vis = &vmexit->u.inout_str;
		rep = vis->inout.rep;
		addrsize = vis->addrsize;
		prot = in ? XHYVE_PROT_WRITE : XHYVE_PROT_READ;
		assert(addrsize == 2 || addrsize == 4 || addrsize == 8);

		/* Index register */
		idxreg = in ? VM_REG_GUEST_RDI : VM_REG_GUEST_RSI;
		index = vis->index & vie_size2mask(addrsize);

		/* Count register */
		count = vis->count & vie_size2mask(addrsize);

		/* Limit number of back-to-back in/out emulations to 16 */
		iterations = min(count, 16);
		while (iterations > 0) {
			assert(retval == 0);
			if (vie_calculate_gla(vis->paging.cpu_mode,
			    vis->seg_name, &vis->seg_desc, index, bytes,
			    addrsize, prot, &gla)) {
				vm_inject_gp(vcpu);
				break;
			}

			error = xh_vm_copy_setup(vcpu, &vis->paging, gla,
			    ((size_t) bytes), prot, iov, nitems(iov), &fault);
			if (error) {
				retval = -1;  /* Unrecoverable error */
				break;
			} else if (fault) {
				retval = 0;  /* Resume guest to handle fault */
				break;
			}

			if (vie_alignment_check(vis->paging.cpl, bytes,
			    vis->cr0, vis->rflags, gla)) {
				vm_inject_ac(vcpu, 0);
				break;
			}

			val = 0;
			if (!in)
				xh_vm_copyin(iov, &val, ((size_t) bytes));

			retval = handler(vcpu, in, port, bytes, &val, arg);
			if (retval != 0)
				break;

			if (in)
				xh_vm_copyout(&val, iov, ((size_t) bytes));

			/* Update index */
			if (vis->rflags & PSL_D)
				index -= ((uint64_t) bytes);
			else
				index += ((uint64_t) bytes);

			count--;
			iterations--;
		}

		/* Update index register */
		error = update_register(vcpu, idxreg, index, addrsize);
		assert(error == 0);

		/*
		 * Update count register only if the instruction had a repeat
		 * prefix.
		 */
		if (rep) {
			error = update_register(vcpu, VM_REG_GUEST_RCX, count, addrsize);
			assert(error == 0);
		}

		/* Restart the instruction if more iterations remain */
		if (retval == 0 && count != 0) {
			error = xh_vm_restart_instruction(vcpu);
			assert(error == 0);
		}
	} else {
		eax = vmexit->u.inout.eax;
		val = eax & vie_size2mask(bytes);
		retval = handler(vcpu, in, port, bytes, &val, arg);
		if (retval == 0 && in) {
			eax &= ~vie_size2mask(bytes);
			eax |= val & vie_size2mask(bytes);
			error = xh_vm_set_register(vcpu, VM_REG_GUEST_RAX, eax);
			assert(error == 0);
		}
	}
	return (retval);
}

void
init_inout(void)
{
	struct inout_port **iopp, *iop;

	/*
	 * Set up the default handler for all ports
	 */
	register_default_iohandler(0, MAX_IOPORTS);

	/*
	 * Overwrite with specified handlers
	 */
	SET_FOREACH(iopp, inout_port_set) {
		iop = *iopp;
		assert(iop->port < MAX_IOPORTS);
		inout_handlers[iop->port].name = iop->name;
		inout_handlers[iop->port].flags = iop->flags;
		inout_handlers[iop->port].handler = iop->handler;
		inout_handlers[iop->port].arg = NULL;
	}
}

int
register_inout(struct inout_port *iop)
{
	int i;

	VERIFY_IOPORT(iop->port, iop->size);

	/*
	 * Verify that the new registration is not overwriting an already
	 * allocated i/o range.
	 */
	if ((((unsigned) iop->flags) & IOPORT_F_DEFAULT) == 0) {
		for (i = iop->port; i < iop->port + iop->size; i++) {
			if ((((unsigned) inout_handlers[i].flags) & IOPORT_F_DEFAULT) == 0)
				return (-1);
		}
	}

	for (i = iop->port; i < iop->port + iop->size; i++) {
		inout_handlers[i].name = iop->name;
		inout_handlers[i].flags = iop->flags;
		inout_handlers[i].handler = iop->handler;
		inout_handlers[i].arg = iop->arg;
	}

	return (0);
}

int
unregister_inout(struct inout_port *iop)
{

	VERIFY_IOPORT(iop->port, iop->size);
	assert(inout_handlers[iop->port].name == iop->name);

	register_default_iohandler(iop->port, iop->size);

	return (0);
}
