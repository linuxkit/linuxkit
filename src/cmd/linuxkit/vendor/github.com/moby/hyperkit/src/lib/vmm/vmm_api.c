/*-
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
 */

#include <stdint.h>
#include <stdbool.h>
#include <string.h>
#include <assert.h>
#include <errno.h>
#include <sys/time.h>
#include <sys/uio.h>
#include <xhyve/support/misc.h>
#include <xhyve/support/specialreg.h>
#include <xhyve/vmm/vmm.h>
#include <xhyve/vmm/vmm_lapic.h>
#include <xhyve/vmm/vmm_instruction_emul.h>
#include <xhyve/vmm/vmm_callout.h>
#include <xhyve/vmm/vmm_stat.h>
#include <xhyve/vmm/vmm_api.h>
#include <xhyve/vmm/io/vatpic.h>
#include <xhyve/vmm/io/vhpet.h>
#include <xhyve/vmm/io/vioapic.h>
#include <xhyve/vmm/io/vrtc.h>

static struct vm *vm;
static int memflags;
static uint32_t lowmem_limit;
static enum vm_mmap_style mmap_style;
static size_t lowmem;
static void *lowmem_addr;
static size_t highmem;
static void *highmem_addr;

static void
vcpu_freeze(int vcpu, bool freeze)
{
	enum vcpu_state state;

	state = (freeze) ? VCPU_FROZEN : VCPU_IDLE;

	if (vcpu_set_state(vm, vcpu, state, freeze)) {
		xhyve_abort("vcpu_set_state failed\n");
	}
}

static void
vcpu_freeze_all(bool freeze)
{
	enum vcpu_state state;
	int vcpu;

	state = (freeze) ? VCPU_FROZEN : VCPU_IDLE;

	for (vcpu = 0; vcpu < VM_MAXCPU; vcpu++) {
		if (vcpu_set_state(vm, vcpu, state, freeze)) {
			xhyve_abort("vcpu_set_state failed\n");
		}
	}
}

void xh_hv_pause(int pause) {
	assert(vm != NULL);
	vm_signal_pause(vm, (pause != 0));
}

int
xh_vm_create(void)
{
	int error;

	if (vm != NULL) {
		return (EEXIST);
	}

	error =	vmm_init();

	if (error != 0) {
		return (error);
	}

	memflags = 0;
	lowmem_limit = (3ull << 30);

	return (vm_create(&vm));
}

void
xh_vm_destroy(void)
{
	assert(vm != NULL);

	vm_destroy(vm);

	if (vmm_cleanup() == 0) {
		vm = NULL;
	}
}

int
xh_vcpu_create(int vcpu)
{
	assert(vm != NULL);
	return (vcpu_create(vm, vcpu));
}

void
xh_vcpu_destroy(int vcpu)
{
	assert(vm != NULL);
	vcpu_destroy(vm, vcpu);
}

int
xh_vm_get_memory_seg(uint64_t gpa, size_t *ret_len)
{
	int error;

	struct vm_memory_segment seg;

	error = vm_gpabase2memseg(vm, gpa, &seg);

	if (error == 0) {
		*ret_len = seg.len;
	}

	return (error);
}

static int
setup_memory_segment(uint64_t gpa, size_t len, void **addr)
{
	void *object;
	uint64_t offset;
	int error;

	vcpu_freeze_all(true);
	error = vm_malloc(vm, gpa, len);
	if (error == 0) {
		error = vm_get_memobj(vm, gpa, len, &offset, &object);
		if (error == 0) {
			*addr = (void *) (((uintptr_t) object) + offset);
		}
	}
	vcpu_freeze_all(false);
	return (error);
}

int
xh_vm_setup_memory(size_t len, enum vm_mmap_style vms)
{
	void **addr;
	int error;

	/* XXX VM_MMAP_SPARSE not implemented yet */
	assert(vms == VM_MMAP_NONE || vms == VM_MMAP_ALL);

	mmap_style = vms;

	/*
	 * If 'len' cannot fit entirely in the 'lowmem' segment then
	 * create another 'highmem' segment above 4GB for the remainder.
	 */

	lowmem = (len > lowmem_limit) ? lowmem_limit : len;
	highmem = (len > lowmem_limit) ? (len - lowmem) : 0;

	if (lowmem > 0) {
		addr = (vms == VM_MMAP_ALL) ? &lowmem_addr : NULL;
		if ((error = setup_memory_segment(0, lowmem, addr))) {
			return (error);
		}
	}

	if (highmem > 0) {
		addr = (vms == VM_MMAP_ALL) ? &highmem_addr : NULL;
		if ((error = setup_memory_segment((4ull << 30), highmem, addr))) {
			return (error);
		}
	}

	return (0);
}

void *
xh_vm_map_gpa(uint64_t gpa, size_t len)
{
	assert(mmap_style == VM_MMAP_ALL);

	if ((gpa < lowmem) && len <= lowmem && ((gpa + len) <= lowmem)) {
		return ((void *) (((uintptr_t) lowmem_addr) + gpa));
	}

	if (gpa >= (4ull << 30)) {
		gpa -= (4ull << 30);
		if ((gpa < highmem) && len <= highmem && ((gpa + len) <= highmem)) {
			return ((void *) (((uintptr_t) highmem_addr) + gpa));
		}
	}

	return (NULL);
}

int
xh_vm_gla2gpa(int vcpu, struct vm_guest_paging *paging, uint64_t gla,
	int prot, uint64_t *gpa, int *fault)
{
	int error;

	vcpu_freeze(vcpu, true);
	error = vm_gla2gpa(vm, vcpu, paging, gla, prot, gpa, fault);
	vcpu_freeze(vcpu, false);

	return (error);
}

uint32_t
xh_vm_get_lowmem_limit(void)
{
	return (lowmem_limit);
}

void
xh_vm_set_lowmem_limit(uint32_t limit)
{
	lowmem_limit = limit;
}

void
xh_vm_set_memflags(int flags)
{
	memflags = flags;
}

size_t
xh_vm_get_lowmem_size(void)
{
	return (lowmem);
}

size_t
xh_vm_get_highmem_size(void)
{
	return (highmem);
}

int
xh_vm_set_desc(int vcpu, int reg, uint64_t base, uint32_t limit,
	uint32_t access)
{
	struct seg_desc sd;
	int error;

	sd.base = base;
	sd.limit = limit;
	sd.access = access;
	vcpu_freeze(vcpu, true);
	error = vm_set_seg_desc(vm, vcpu, reg, &sd);
	vcpu_freeze(vcpu, false);

	return (error);
}

int
xh_vm_get_desc(int vcpu, int reg, uint64_t *base, uint32_t *limit,
	uint32_t *access)
{
	struct seg_desc sd;
	int error;

	vcpu_freeze(vcpu, true);
	error = vm_get_seg_desc(vm, vcpu, reg, &sd);
	if (error == 0) {
		*base = sd.base;
		*limit = sd.limit;
		*access = sd.access;
	}
	vcpu_freeze(vcpu, false);

	return (error);
}

int
xh_vm_get_seg_desc(int vcpu, int reg, struct seg_desc *seg_desc)
{
	int error;

	error = xh_vm_get_desc(vcpu, reg, &seg_desc->base, &seg_desc->limit,
		&seg_desc->access);

	return (error);
}

int
xh_vm_set_register(int vcpu, int reg, uint64_t val)
{
	int error;

	vcpu_freeze(vcpu, true);
	error = vm_set_register(vm, vcpu, reg, val);
	vcpu_freeze(vcpu, false);

	return (error);
}

int
xh_vm_get_register(int vcpu, int reg, uint64_t *retval)
{
	int error;

	vcpu_freeze(vcpu, true);
	error = vm_get_register(vm, vcpu, reg, retval);
	vcpu_freeze(vcpu, false);

	return (error);
}

int
xh_vm_run(int vcpu, struct vm_exit *ret_vmexit)
{
	int error;

	vcpu_freeze(vcpu, true);
	error = vm_run(vm, vcpu, ret_vmexit);
	vcpu_freeze(vcpu, false);

	return (error);
}

int
xh_vm_suspend(enum vm_suspend_how how)
{
	return (vm_suspend(vm, how));
}

int
xh_vm_reinit(void)
{
	int error;

	vcpu_freeze_all(true);
	error = vm_reinit(vm);
	vcpu_freeze_all(false);

	return (error);
}

int
xh_vm_apicid2vcpu(int apicid)
{
	return (apicid);
}

int
xh_vm_inject_exception(int vcpu, int vector, int errcode_valid,
	uint32_t errcode, int restart_instruction)
{
	int error;

	vcpu_freeze(vcpu, true);
	error = vm_inject_exception(vm, vcpu, vector, errcode_valid, errcode,
		restart_instruction);
	vcpu_freeze(vcpu, false);

	return (error);
}

int
xh_vm_lapic_irq(int vcpu, int vector)
{
	return (lapic_intr_edge(vm, vcpu, vector));
}

int
xh_vm_lapic_local_irq(int vcpu, int vector)
{
	return (lapic_set_local_intr(vm, vcpu, vector));
}

int
xh_vm_lapic_msi(uint64_t addr, uint64_t msg)
{
	return (lapic_intr_msi(vm, addr, msg));
}

int
xh_vm_ioapic_assert_irq(int irq)
{
	return (vioapic_assert_irq(vm, irq));
}

int
xh_vm_ioapic_deassert_irq(int irq)
{
	return (vioapic_deassert_irq(vm, irq));
}

int
xh_vm_ioapic_pulse_irq(int irq)
{
	return (vioapic_pulse_irq(vm, irq));
}

int
xh_vm_ioapic_pincount(int *pincount)
{
	*pincount = vioapic_pincount(vm);
	return (0);
}

int
xh_vm_isa_assert_irq(int atpic_irq, int ioapic_irq)
{
	int error;

	error = vatpic_assert_irq(vm, atpic_irq);

	if ((error == 0) && (ioapic_irq != -1)) {
		error = vioapic_assert_irq(vm, ioapic_irq);
	}

	return (error);
}

int
xh_vm_isa_deassert_irq(int atpic_irq, int ioapic_irq)
{
	int error;

	error = vatpic_deassert_irq(vm, atpic_irq);
	if ((error == 0) && (ioapic_irq != -1)) {
		error = vioapic_deassert_irq(vm, ioapic_irq);
	}

	return (error);
}

int
xh_vm_isa_pulse_irq(int atpic_irq, int ioapic_irq)
{
	int error;

	error = vatpic_pulse_irq(vm, atpic_irq);
	if ((error == 0) && (ioapic_irq != -1)) {
		error = vioapic_pulse_irq(vm, ioapic_irq);
	}

	return (error);
}

int
xh_vm_isa_set_irq_trigger(int atpic_irq, enum vm_intr_trigger trigger)
{
	return (vatpic_set_irq_trigger(vm, atpic_irq, trigger));
}

int
xh_vm_inject_nmi(int vcpu)
{
	return (vm_inject_nmi(vm, vcpu));
}

static struct {
	const char *name;
	int type;
} capstrmap[] = {
	{ "hlt_exit", VM_CAP_HALT_EXIT },
	{ "mtrap_exit", VM_CAP_MTRAP_EXIT },
	{ "pause_exit", VM_CAP_PAUSE_EXIT },
	{ NULL, 0 }
};

int
xh_vm_capability_name2type(const char *capname)
{
	int i;

	for (i = 0; (capstrmap[i].name != NULL) && (capname != NULL); i++) {
		if (strcmp(capstrmap[i].name, capname) == 0) {
			return (capstrmap[i].type);
		}
	}

	return (-1);
}

const char *
xh_vm_capability_type2name(int type)
{
	int i;

	for (i = 0; (capstrmap[i].name != NULL); i++) {
		if (capstrmap[i].type == type) {
			return (capstrmap[i].name);
		}
	}

	return (NULL);
}

int
xh_vm_get_capability(int vcpu, enum vm_cap_type cap, int *retval)
{
	int error;

	vcpu_freeze(vcpu, true);
	error = vm_get_capability(vm, vcpu, cap, retval);
	vcpu_freeze(vcpu, false);

	return (error);
}

int
xh_vm_set_capability(int vcpu, enum vm_cap_type cap, int val)
{
	int error;

	vcpu_freeze(vcpu, true);
	error = vm_set_capability(vm, vcpu, cap, val);
	vcpu_freeze(vcpu, false);

	return (error);
}

int
xh_vm_get_intinfo(int vcpu, uint64_t *i1, uint64_t *i2)
{
	int error;

	vcpu_freeze(vcpu, true);
	error = vm_get_intinfo(vm, vcpu, i1, i2);
	vcpu_freeze(vcpu, false);

	return (error);
}

int
xh_vm_set_intinfo(int vcpu, uint64_t exit_intinfo)
{
	int error;

	vcpu_freeze(vcpu, true);
	error = vm_exit_intinfo(vm, vcpu, exit_intinfo);
	vcpu_freeze(vcpu, false);

	return (error);
}

uint64_t *
xh_vm_get_stats(int vcpu, struct timeval *ret_tv, int *ret_entries)
{
	static uint64_t statbuf[64];
	struct timeval tv;
	int re;
	int error;

	getmicrotime(&tv);
	error = vmm_stat_copy(vm, vcpu, &re, ((uint64_t *) &statbuf));

	if (error == 0) {
		if (ret_entries) {
			*ret_entries = re;
		}
		if (ret_tv) {
			*ret_tv = tv;
		}
		return (((uint64_t *) &statbuf));
	} else {
		return (NULL);
	}
}

const char *
xh_vm_get_stat_desc(int index)
{
	static char desc[128];

	if (vmm_stat_desc_copy(index, ((char *) &desc), sizeof(desc)) == 0) {
		return (desc);
	} else {
		return (NULL);
	}
}

int
xh_vm_get_x2apic_state(int vcpu, enum x2apic_state *s)
{
	return (vm_get_x2apic_state(vm, vcpu, s));
}

int
xh_vm_set_x2apic_state(int vcpu, enum x2apic_state s)
{
	int error;

	vcpu_freeze(vcpu, true);
	error = vm_set_x2apic_state(vm, vcpu, s);
	vcpu_freeze(vcpu, false);

	return (error);
}

int
xh_vm_get_hpet_capabilities(uint32_t *capabilities)
{
	return (vhpet_getcap(capabilities));
}

int
xh_vm_copy_setup(int vcpu, struct vm_guest_paging *pg, uint64_t gla, size_t len,
	int prot, struct iovec *iov, int iovcnt, int *fault)
{
	void *va;
	uint64_t gpa;
	size_t n, off;
	int i, error;

	for (i = 0; i < iovcnt; i++) {
		iov[i].iov_base = 0;
		iov[i].iov_len = 0;
	}

	while (len) {
		assert(iovcnt > 0);

		error = xh_vm_gla2gpa(vcpu, pg, gla, prot, &gpa, fault);
		if ((error) || *fault) {
			return (error);
		}

		off = gpa & XHYVE_PAGE_MASK;
		n = min(len, XHYVE_PAGE_SIZE - off);

		va = xh_vm_map_gpa(gpa, n);
		if (va == NULL) {
			return (EFAULT);
		}

		iov->iov_base = va;
		iov->iov_len = n;
		iov++;
		iovcnt--;

		gla += n;
		len -= n;
	}

	return (0);
}

void
xh_vm_copyin(struct iovec *iov, void *dst, size_t len)
{
	const char *src;
	char *d;
	size_t n;

	d = dst;
	while (len) {
		assert(iov->iov_len);
		n = min(len, iov->iov_len);
		src = iov->iov_base;
		bcopy(src, d, n);
		iov++;
		d += n;
		len -= n;
	}
}

void
xh_vm_copyout(const void *src, struct iovec *iov, size_t len)
{
	const char *s;
	char *dst;
	size_t n;

	s = src;
	while (len) {
		assert(iov->iov_len);
		n = min(len, iov->iov_len);
		dst = iov->iov_base;
		bcopy(s, dst, n);
		iov++;
		s += n;
		len -= n;
	}
}

int
xh_vm_rtc_write(int offset, uint8_t value)
{
	return (vrtc_nvram_write(vm, offset, value));
}

int
xh_vm_rtc_read(int offset, uint8_t *retval)
{
	return (vrtc_nvram_read(vm, offset, retval));
}

int
xh_vm_rtc_settime(time_t secs)
{
	return (vrtc_set_time(vm, secs));
}

int
xh_vm_rtc_gettime(time_t *secs)
{
	*secs = vrtc_get_time(vm);
	return (0);
}

int
xh_vcpu_reset(int vcpu)
{
	int error;

#define SET_REG(r, v) (error = xh_vm_set_register(vcpu, (r), (v)))
#define SET_DESC(d, b, l, a) (error = xh_vm_set_desc(vcpu, (d), (b), (l), (a)))

	if (SET_REG(VM_REG_GUEST_RFLAGS, 0x2) ||
		SET_REG(VM_REG_GUEST_RIP, 0xfff0) ||
		SET_REG(VM_REG_GUEST_CR0, CR0_NE) ||
		SET_REG(VM_REG_GUEST_CR3, 0) ||
		SET_REG(VM_REG_GUEST_CR4, 0) ||
		SET_REG(VM_REG_GUEST_CS, 0xf000) ||
		SET_REG(VM_REG_GUEST_SS, 0) ||
		SET_REG(VM_REG_GUEST_DS, 0) ||
		SET_REG(VM_REG_GUEST_ES, 0) ||
		SET_REG(VM_REG_GUEST_FS, 0) ||
		SET_REG(VM_REG_GUEST_GS, 0) ||
		SET_REG(VM_REG_GUEST_RAX, 0) ||
		SET_REG(VM_REG_GUEST_RBX, 0) ||
		SET_REG(VM_REG_GUEST_RCX, 0) ||
		SET_REG(VM_REG_GUEST_RDX, 0xf00) ||
		SET_REG(VM_REG_GUEST_RSI, 0) ||
		SET_REG(VM_REG_GUEST_RDI, 0) ||
		SET_REG(VM_REG_GUEST_RBP, 0) ||
		SET_REG(VM_REG_GUEST_RSP, 0) ||
		SET_REG(VM_REG_GUEST_TR, 0) ||
		SET_REG(VM_REG_GUEST_LDTR, 0) ||
		SET_DESC(VM_REG_GUEST_CS, 0xffff0000, 0xffff, 0x0093) ||
		SET_DESC(VM_REG_GUEST_SS, 0, 0xffff, 0x0093) ||
		SET_DESC(VM_REG_GUEST_DS, 0, 0xffff, 0x0093) ||
		SET_DESC(VM_REG_GUEST_ES, 0, 0xffff, 0x0093) ||
		SET_DESC(VM_REG_GUEST_FS, 0, 0xffff, 0x0093) ||
		SET_DESC(VM_REG_GUEST_GS, 0, 0xffff, 0x0093) ||
		SET_DESC(VM_REG_GUEST_GDTR, 0, 0xffff, 0) ||
		SET_DESC(VM_REG_GUEST_IDTR, 0, 0xffff, 0) ||
		SET_DESC(VM_REG_GUEST_TR, 0, 0, 0x0000008b) ||
		SET_DESC(VM_REG_GUEST_LDTR, 0, 0xffff, 0x00000082))
	{
		return (error);
	}

	return (0);
}

int
xh_vm_active_cpus(cpuset_t *cpus)
{
	*cpus = vm_active_cpus(vm);
	return (0);
}

int
xh_vm_suspended_cpus(cpuset_t *cpus)
{
	*cpus = vm_suspended_cpus(vm);
	return (0);
}

int
xh_vm_activate_cpu(int vcpu)
{
	int error;

	vcpu_freeze(vcpu, true);
	error = vm_activate_cpu(vm, vcpu);
	vcpu_freeze(vcpu, false);

	return (error);
}

int
xh_vm_restart_instruction(int vcpu)
{
	int error;

	vcpu_freeze(vcpu, true);
	error = vm_restart_instruction(vm, vcpu);
	vcpu_freeze(vcpu, false);

	return (error);
}

int
xh_vm_emulate_instruction(int vcpu, uint64_t gpa, struct vie *vie,
	struct vm_guest_paging *paging, mem_region_read_t memread,
	mem_region_write_t memwrite, void *memarg)
{
	int error;

	vcpu_freeze(vcpu, true);
	error = vmm_emulate_instruction(vm, vcpu, gpa, vie, paging, memread,
		memwrite, memarg);
	vcpu_freeze(vcpu, false);

	return (error);
}

void
xh_vm_vcpu_dump(int vcpu)
{
	vm_vcpu_dump(vm, vcpu);
}
