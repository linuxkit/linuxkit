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

#include <stdint.h>
#include <stdbool.h>
#include <string.h>
#include <errno.h>
#include <pthread.h>
#include <assert.h>
#include <libkern/OSAtomic.h>
#include <xhyve/support/misc.h>
#include <xhyve/support/atomic.h>
#include <xhyve/support/cpuset.h>
#include <xhyve/support/psl.h>
#include <xhyve/support/specialreg.h>
#include <xhyve/support/apicreg.h>
#include <xhyve/vmm/vmm.h>
#include <xhyve/vmm/vmm_lapic.h>
#include <xhyve/vmm/vmm_mem.h>
#include <xhyve/vmm/vmm_ioport.h>
#include <xhyve/vmm/vmm_instruction_emul.h>
#include <xhyve/vmm/vmm_callout.h>
#include <xhyve/vmm/vmm_host.h>
#include <xhyve/vmm/vmm_stat.h>
#include <xhyve/vmm/vmm_ktr.h>
#include <xhyve/vmm/io/vatpic.h>
#include <xhyve/vmm/io/vatpit.h>
#include <xhyve/vmm/io/vioapic.h>
#include <xhyve/vmm/io/vlapic.h>
#include <xhyve/vmm/io/vhpet.h>
#include <xhyve/vmm/io/vpmtmr.h>
#include <xhyve/vmm/io/vrtc.h>

struct vlapic;

/*
 * Initialization:
 * (a) allocated when vcpu is created
 * (i) initialized when vcpu is created and when it is reinitialized
 * (o) initialized the first time the vcpu is created
 * (x) initialized before use
 */
struct vcpu {
	pthread_mutex_t lock; /* (o) protects 'state' */
	pthread_mutex_t state_sleep_mtx;
	pthread_cond_t state_sleep_cnd;
	pthread_mutex_t vcpu_sleep_mtx;
	pthread_cond_t vcpu_sleep_cnd;
	enum vcpu_state state; /* (o) vcpu state */
	struct vlapic *vlapic; /* (i) APIC device model */
	enum x2apic_state x2apic_state; /* (i) APIC mode */
	uint64_t exitintinfo; /* (i) events pending at VM exit */
	int nmi_pending; /* (i) NMI pending */
	int extint_pending; /* (i) INTR pending */
	int exception_pending; /* (i) exception pending */
	int exc_vector; /* (x) exception collateral */
	int exc_errcode_valid;
	uint32_t exc_errcode;
	uint64_t guest_xcr0; /* (i) guest %xcr0 register */
	void *stats; /* (a,i) statistics */
	struct vm_exit exitinfo; /* (x) exit reason and collateral */
	uint64_t nextrip; /* (x) next instruction to execute */
};

#define vcpu_lock_init(v) xpthread_mutex_init(&(v)->lock)
#define vcpu_lock(v) xpthread_mutex_lock(&(v)->lock)
#define vcpu_unlock(v) xpthread_mutex_unlock(&(v)->lock)

struct mem_seg {
	uint64_t gpa;
	size_t len;
	void *object;
};

#define	VM_MAX_MEMORY_SEGMENTS	2

/*
 * Initialization:
 * (o) initialized the first time the VM is created
 * (i) initialized when VM is created and when it is reinitialized
 * (x) initialized before use
 */
struct vm {
	void *cookie; /* (i) cpu-specific data */
	struct vhpet *vhpet; /* (i) virtual HPET */
	struct vioapic *vioapic; /* (i) virtual ioapic */
	struct vatpic *vatpic; /* (i) virtual atpic */
	struct vatpit *vatpit; /* (i) virtual atpit */
	struct vpmtmr *vpmtmr; /* (i) virtual ACPI PM timer */
	struct vrtc *vrtc; /* (o) virtual RTC */
	volatile cpuset_t active_cpus; /* (i) active vcpus */
	int suspend; /* (i) stop VM execution */
	volatile cpuset_t suspended_cpus; /* (i) suspended vcpus */
	volatile cpuset_t halted_cpus; /* (x) cpus in a hard halt */
	cpuset_t rendezvous_req_cpus; /* (x) rendezvous requested */
	cpuset_t rendezvous_done_cpus; /* (x) rendezvous finished */
	void *rendezvous_arg; /* (x) rendezvous func/arg */
	vm_rendezvous_func_t rendezvous_func;
	pthread_mutex_t rendezvous_mtx; /* (o) rendezvous lock */
	pthread_cond_t rendezvous_sleep_cnd;
	int num_mem_segs; /* (o) guest memory segments */
	struct mem_seg mem_segs[VM_MAX_MEMORY_SEGMENTS];
	struct vcpu vcpu[VM_MAXCPU]; /* (i) guest vcpus */
	volatile u_int hv_is_paused;
	pthread_mutex_t hv_pause_mtx;
	pthread_cond_t hv_pause_cnd;
};

static int vmm_initialized;

static struct vmm_ops *ops;

#define	VMM_INIT() \
	(*ops->init)()
#define	VMM_CLEANUP() \
	(*ops->cleanup)()
#define	VM_INIT(vmi) \
	(*ops->vm_init)(vmi)
#define	VCPU_INIT(vmi, vcpu) \
	(*ops->vcpu_init)(vmi, vcpu)
#define	VMRUN(vmi, vcpu, rip, rptr, sptr) \
	(*ops->vmrun)(vmi, vcpu, rip, rptr, sptr)
#define VCPU_DUMP(vmi, vcpu) \
	(*ops->vcpu_dump)(vmi, vcpu)
#define	VM_CLEANUP(vmi) \
	(*ops->vm_cleanup)(vmi)
#define	VCPU_CLEANUP(vmi, vcpu) \
	(*ops->vcpu_cleanup)(vmi, vcpu)
#define	VMGETREG(vmi, vcpu, num, retval) \
	(*ops->vmgetreg)(vmi, vcpu, num, retval)
#define	VMSETREG(vmi, vcpu, num, val) \
	(*ops->vmsetreg)(vmi, vcpu, num, val)
#define	VMGETDESC(vmi, vcpu, num, desc) \
	(*ops->vmgetdesc)(vmi, vcpu, num, desc)
#define	VMSETDESC(vmi, vcpu, num, desc) \
	(*ops->vmsetdesc)(vmi, vcpu, num, desc)
#define	VMGETCAP(vmi, vcpu, num, retval) \
	(*ops->vmgetcap)(vmi, vcpu, num, retval)
#define	VMSETCAP(vmi, vcpu, num, val) \
	(*ops->vmsetcap)(vmi, vcpu, num, val)
#define	VLAPIC_INIT(vmi, vcpu) \
	(*ops->vlapic_init)(vmi, vcpu)
#define	VLAPIC_CLEANUP(vmi, vlapic) \
	(*ops->vlapic_cleanup)(vmi, vlapic)
#define	VCPU_INTERRUPT(vcpu) \
	(*ops->vcpu_interrupt)(vcpu)

/* statistics */
//static VMM_STAT(VCPU_TOTAL_RUNTIME, "vcpu total runtime");

/*
 * Halt the guest if all vcpus are executing a HLT instruction with
 * interrupts disabled.
 */
static int halt_detection_enabled = 1;
static int trace_guest_exceptions = 0;

static void
vcpu_cleanup(struct vm *vm, int i, bool destroy)
{
	struct vcpu *vcpu = &vm->vcpu[i];

	VLAPIC_CLEANUP(vm->cookie, vcpu->vlapic);
	if (destroy) {
		vmm_stat_free(vcpu->stats);
	}
}

static void
vcpu_init(struct vm *vm, int vcpu_id, bool create)
{
	struct vcpu *vcpu;

	KASSERT(vcpu_id >= 0 && vcpu_id < VM_MAXCPU,
	    ("vcpu_init: invalid vcpu %d", vcpu_id));

	vcpu = &vm->vcpu[vcpu_id];

	if (create) {
		vcpu_lock_init(vcpu);
		pthread_mutex_init(&vcpu->state_sleep_mtx, NULL);
		pthread_cond_init(&vcpu->state_sleep_cnd, NULL);
		pthread_mutex_init(&vcpu->vcpu_sleep_mtx, NULL);
		pthread_cond_init(&vcpu->vcpu_sleep_cnd, NULL);
		vcpu->state = VCPU_IDLE;
		vcpu->stats = vmm_stat_alloc();
	}

	vcpu->vlapic = VLAPIC_INIT(vm->cookie, vcpu_id);
	vm_set_x2apic_state(vm, vcpu_id, X2APIC_DISABLED);
	vcpu->exitintinfo = 0;
	vcpu->nmi_pending = 0;
	vcpu->extint_pending = 0;
	vcpu->exception_pending = 0;
	vcpu->guest_xcr0 = XFEATURE_ENABLED_X87;
	vmm_stat_init(vcpu->stats);
}

int vcpu_create(struct vm *vm, int vcpu) {
	if (vcpu < 0 || vcpu >= VM_MAXCPU)
		xhyve_abort("vcpu_create: invalid cpuid %d\n", vcpu);

	return VCPU_INIT(vm->cookie, vcpu);
}

void vcpu_destroy(struct vm *vm, int vcpu) {
	if (vcpu < 0 || vcpu >= VM_MAXCPU)
		xhyve_abort("vcpu_destroy: invalid cpuid %d\n", vcpu);

	VCPU_CLEANUP(vm, vcpu);
}

int
vcpu_trace_exceptions(void)
{
	return (trace_guest_exceptions);
}

struct vm_exit *
vm_exitinfo(struct vm *vm, int cpuid)
{
	struct vcpu *vcpu;

	if (cpuid < 0 || cpuid >= VM_MAXCPU)
		xhyve_abort("vm_exitinfo: invalid cpuid %d\n", cpuid);

	vcpu = &vm->vcpu[cpuid];

	return (&vcpu->exitinfo);
}

int
vmm_init(void)
{
	int error;

	vmm_host_state_init();

	error = vmm_mem_init();
	if (error)
		return (error);

	ops = &vmm_ops_intel;

	error = VMM_INIT();

	if (error == 0)
		vmm_initialized = 1;

	return (error);
}


int
vmm_cleanup(void) {
	int error;

	error = VMM_CLEANUP();

	if (error == 0)
		vmm_initialized = 0;

	return error;
}

static void
vm_init(struct vm *vm, bool create)
{
	int vcpu;

	if (create) {
		callout_system_init();
	}

	vm->cookie = VM_INIT(vm);
	vm->vioapic = vioapic_init(vm);
	vm->vhpet = vhpet_init(vm);
	vm->vatpic = vatpic_init(vm);
	vm->vatpit = vatpit_init(vm);
	vm->vpmtmr = vpmtmr_init(vm);

	if (create) {
		vm->vrtc = vrtc_init(vm);
	}

	CPU_ZERO(&vm->active_cpus);

	vm->suspend = 0;
	CPU_ZERO(&vm->suspended_cpus);

	for (vcpu = 0; vcpu < VM_MAXCPU; vcpu++) {
		vcpu_init(vm, vcpu, create);
	}
}

int
vm_create(struct vm **retvm)
{
	struct vm *vm;

	if (!vmm_initialized)
		return (ENXIO);

	vm = malloc(sizeof(struct vm));
	assert(vm);
	bzero(vm, sizeof(struct vm));
	vm->num_mem_segs = 0;
	pthread_mutex_init(&vm->rendezvous_mtx, NULL);
	pthread_cond_init(&vm->rendezvous_sleep_cnd, NULL);

	vm->hv_is_paused = FALSE;
	pthread_mutex_init(&vm->hv_pause_mtx, NULL);
	pthread_cond_init(&vm->hv_pause_cnd, NULL);

	vm_init(vm, true);

	*retvm = vm;
	return (0);
}

static void
vm_mem_protect(struct vm *vm) {
	for (int i = 0; i < vm->num_mem_segs; i++) {
		vmm_mem_protect(vm->mem_segs[i].gpa, vm->mem_segs[i].len);
	}
}

static void
vm_mem_unprotect(struct vm *vm) {
	for (int i = 0; i < vm->num_mem_segs; i++) {
		vmm_mem_unprotect(vm->mem_segs[i].gpa, vm->mem_segs[i].len);
	}
}

void
vm_signal_pause(struct vm *vm, bool pause) {
	pthread_mutex_lock(&vm->hv_pause_mtx); // Lock as we are modifying hv_is_paused
	if (pause) {
		if (atomic_cmpset_rel_int(&vm->hv_is_paused, FALSE, TRUE) == 0) { // All vcpus should wait after next interrupt
			fprintf(stderr, "freeze signal received, but we are already frozen\n");
		}
	} else {
		if (atomic_cmpset_rel_int(&vm->hv_is_paused, TRUE, FALSE) == 0) {
			fprintf(stderr, "thaw signal received, but we are not frozen\n");
		} else {
			pthread_cond_broadcast(&vm->hv_pause_cnd); // wake paused threads
		}
	}
	pthread_mutex_unlock(&vm->hv_pause_mtx);
}


static bool vm_is_paused(struct vm *vm) {
	return atomic_load_acq_int(&vm->hv_is_paused) == TRUE;
}

void
vm_check_for_unpause(struct vm *vm, int vcpuid) {
	enum vcpu_state state;
	if (vm_is_paused(vm)) {
		if (pthread_mutex_lock(&vm->hv_pause_mtx) != 0) {
			xhyve_abort("error locking mutex");
		}
		if (vm_is_paused(vm)) { // Check that we are still paused after acq lock
			enum vcpu_state orig_state = vm->vcpu[vcpuid].state;
			state = VCPU_FROZEN;
			if (vcpu_set_state(vm, vcpuid, state, false) != 0) {
				xhyve_abort("vcpu_set_state failed\n");
			}
			vm_mem_protect(vm);

			// Wait for signal
			fprintf(stderr, "vcpu %d waiting for signal to resume\n", vcpuid);
			do {
				if (pthread_cond_wait(&vm->hv_pause_cnd, &vm->hv_pause_mtx) != 0) {
					xhyve_abort("pthread_cond_wait failed");
				}
			} while (vm_is_paused(vm));
			fprintf(stderr, "vcpu %d received signal, resuming\n", vcpuid);

			vm_mem_unprotect(vm);
			state = orig_state;
			if (vcpu_set_state(vm, vcpuid, state, false) != 0) {
				xhyve_abort("vcpu_set_state failed\n");
			}
		}
		if (pthread_mutex_unlock(&vm->hv_pause_mtx) != 0) {
			xhyve_abort("mutex unlock failed");
		}
	}
}

static void
vm_free_mem_seg(struct mem_seg *seg)
{
	if (seg->object != NULL) {
		vmm_mem_free(seg->gpa, seg->len, seg->object);
	}

	bzero(seg, sizeof(*seg));
}

static void
vm_cleanup(struct vm *vm, bool destroy)
{
	int i, vcpu;

	for (vcpu = 0; vcpu < VM_MAXCPU; vcpu++) {
		vcpu_cleanup(vm, vcpu, destroy);
	}

	if (destroy) {
		vrtc_cleanup(vm->vrtc);
	} else {
		vrtc_reset(vm->vrtc);
	}
	vpmtmr_cleanup(vm->vpmtmr);
	vatpit_cleanup(vm->vatpit);
	vhpet_cleanup(vm->vhpet);
	vatpic_cleanup(vm->vatpic);
	vioapic_cleanup(vm->vioapic);

	VM_CLEANUP(vm->cookie);

	if (destroy) {
		for (i = 0; i < vm->num_mem_segs; i++) {
			vm_free_mem_seg(&vm->mem_segs[i]);
		}

		vm->num_mem_segs = 0;
	}
}

void
vm_destroy(struct vm *vm)
{
	vm_cleanup(vm, true);
	free(vm);
}

int
vm_reinit(struct vm *vm)
{
	int error;

	/*
	 * A virtual machine can be reset only if all vcpus are suspended.
	 */
	if (CPU_CMP(&vm->suspended_cpus, &vm->active_cpus) == 0) {
		vm_cleanup(vm, false);
		vm_init(vm, false);
		error = 0;
	} else {
		error = EBUSY;
	}

	return (error);
}

const char *
vm_name(UNUSED struct vm *vm)
{
	return "VM";
}

bool
vm_mem_allocated(struct vm *vm, uint64_t gpa)
{
	int i;
	uint64_t gpabase, gpalimit;

	for (i = 0; i < vm->num_mem_segs; i++) {
		gpabase = vm->mem_segs[i].gpa;
		gpalimit = gpabase + vm->mem_segs[i].len;
		if (gpa >= gpabase && gpa < gpalimit)
			return (TRUE);		/* 'gpa' is regular memory */
	}

	return (FALSE);
}

int
vm_malloc(struct vm *vm, uint64_t gpa, size_t len)
{
	int available, allocated;
	struct mem_seg *seg;
	void *object;
	uint64_t g;

	if ((gpa & XHYVE_PAGE_MASK) || (len & XHYVE_PAGE_MASK) || len == 0)
		return (EINVAL);

	available = allocated = 0;
	g = gpa;
	while (g < gpa + len) {
		if (vm_mem_allocated(vm, g))
			allocated++;
		else
			available++;

		g += XHYVE_PAGE_SIZE;
	}

	/*
	 * If there are some allocated and some available pages in the address
	 * range then it is an error.
	 */
	if (allocated && available)
		return (EINVAL);

	/*
	 * If the entire address range being requested has already been
	 * allocated then there isn't anything more to do.
	 */
	if (allocated && available == 0)
		return (0);

	if (vm->num_mem_segs >= VM_MAX_MEMORY_SEGMENTS)
		return (E2BIG);

	seg = &vm->mem_segs[vm->num_mem_segs];

	if ((object = vmm_mem_alloc(gpa, len)) == NULL)
		return (ENOMEM);

	seg->gpa = gpa;
	seg->len = len;
	seg->object = object;

	vm->num_mem_segs++;

	return (0);
}

void *
vm_gpa2hva(struct vm *vm, uint64_t gpa, uint64_t len) {
	void *base;
	uint64_t offset;

	if (vm_get_memobj(vm, gpa, len, &offset, &base)) {
		return NULL;
	}

	return (void *) (((uintptr_t) base) + offset);
}

int
vm_gpabase2memseg(struct vm *vm, uint64_t gpabase,
		  struct vm_memory_segment *seg)
{
	int i;

	for (i = 0; i < vm->num_mem_segs; i++) {
		if (gpabase == vm->mem_segs[i].gpa) {
			seg->gpa = vm->mem_segs[i].gpa;
			seg->len = vm->mem_segs[i].len;
			return (0);
		}
	}
	return (-1);
}

int
vm_get_memobj(struct vm *vm, uint64_t gpa, size_t len,
	      uint64_t *offset, void **object)
{
	int i;
	size_t seg_len;
	uint64_t seg_gpa;
	void *seg_obj;

	for (i = 0; i < vm->num_mem_segs; i++) {
		if ((seg_obj = vm->mem_segs[i].object) == NULL)
			continue;

		seg_gpa = vm->mem_segs[i].gpa;
		seg_len = vm->mem_segs[i].len;

		if ((gpa >= seg_gpa) && ((gpa + len) <= (seg_gpa + seg_len))) {
			*offset = gpa - seg_gpa;
			*object = seg_obj;
			return (0);
		}
	}

	return (EINVAL);
}

int
vm_get_register(struct vm *vm, int vcpu, int reg, uint64_t *retval)
{

	if (vcpu < 0 || vcpu >= VM_MAXCPU)
		return (EINVAL);

	if (reg >= VM_REG_LAST)
		return (EINVAL);

	return (VMGETREG(vm->cookie, vcpu, reg, retval));
}

int
vm_set_register(struct vm *vm, int vcpuid, int reg, uint64_t val)
{
	struct vcpu *vcpu;
	int error;

	if (vcpuid < 0 || vcpuid >= VM_MAXCPU)
		return (EINVAL);

	if (reg >= VM_REG_LAST)
		return (EINVAL);

	error = VMSETREG(vm->cookie, vcpuid, reg, val);
	if (error || reg != VM_REG_GUEST_RIP)
		return (error);

	/* Set 'nextrip' to match the value of %rip */
	VCPU_CTR1(vm, vcpuid, "Setting nextrip to %#llx", val);
	vcpu = &vm->vcpu[vcpuid];
	vcpu->nextrip = val;
	return (0);
}

static bool
is_descriptor_table(int reg)
{
	switch (reg) {
	case VM_REG_GUEST_IDTR:
	case VM_REG_GUEST_GDTR:
		return (TRUE);
	default:
		return (FALSE);
	}
}

static bool
is_segment_register(int reg)
{
	switch (reg) {
	case VM_REG_GUEST_ES:
	case VM_REG_GUEST_CS:
	case VM_REG_GUEST_SS:
	case VM_REG_GUEST_DS:
	case VM_REG_GUEST_FS:
	case VM_REG_GUEST_GS:
	case VM_REG_GUEST_TR:
	case VM_REG_GUEST_LDTR:
		return (TRUE);
	default:
		return (FALSE);
	}
}

int
vm_get_seg_desc(struct vm *vm, int vcpu, int reg,
		struct seg_desc *desc)
{
	if (vcpu < 0 || vcpu >= VM_MAXCPU)
		return (EINVAL);

	if (!is_segment_register(reg) && !is_descriptor_table(reg))
		return (EINVAL);

	return (VMGETDESC(vm->cookie, vcpu, reg, desc));
}

int
vm_set_seg_desc(struct vm *vm, int vcpu, int reg,
		struct seg_desc *desc)
{
	if (vcpu < 0 || vcpu >= VM_MAXCPU)
		return (EINVAL);

	if (!is_segment_register(reg) && !is_descriptor_table(reg))
		return (EINVAL);

	return (VMSETDESC(vm->cookie, vcpu, reg, desc));
}

// static VMM_STAT(VCPU_IDLE_TICKS, "number of ticks vcpu was idle");

static int
vcpu_set_state_locked(struct vcpu *vcpu, enum vcpu_state newstate,
    bool from_idle)
{
	int error;
	const struct timespec ts = {.tv_sec = 1, .tv_nsec = 0}; /* 1 second */

	/*
	 * State transitions from the vmmdev_ioctl() must always begin from
	 * the VCPU_IDLE state. This guarantees that there is only a single
	 * ioctl() operating on a vcpu at any point.
	 */
	if (from_idle) {
		while (vcpu->state != VCPU_IDLE) {
			pthread_mutex_lock(&vcpu->state_sleep_mtx);
			vcpu_unlock(vcpu);
			pthread_cond_timedwait_relative_np(&vcpu->state_sleep_cnd,
				&vcpu->state_sleep_mtx, &ts);
			vcpu_lock(vcpu);
			pthread_mutex_unlock(&vcpu->state_sleep_mtx);
			//msleep_spin(&vcpu->state, &vcpu->mtx, "vmstat", hz);
		}
	} else {
		KASSERT(vcpu->state != VCPU_IDLE, ("invalid transition from "
		    "vcpu idle state"));
	}

	/*
	 * The following state transitions are allowed:
	 * IDLE -> FROZEN -> IDLE
	 * FROZEN -> RUNNING -> FROZEN
	 * FROZEN -> SLEEPING -> FROZEN
	 */
	switch (vcpu->state) {
	case VCPU_IDLE:
	case VCPU_RUNNING:
	case VCPU_SLEEPING:
		error = (newstate != VCPU_FROZEN);
		break;
	case VCPU_FROZEN:
		error = (newstate == VCPU_FROZEN);
		break;
	}

	if (error)
		return (EBUSY);

	vcpu->state = newstate;

	if (newstate == VCPU_IDLE)
		pthread_cond_broadcast(&vcpu->state_sleep_cnd);
		//wakeup(&vcpu->state);

	return (0);
}

static void
vcpu_require_state(struct vm *vm, int vcpuid, enum vcpu_state newstate)
{
	int error;

	if ((error = vcpu_set_state(vm, vcpuid, newstate, false)) != 0)
		xhyve_abort("Error %d setting state to %d\n", error, newstate);
}

static void
vcpu_require_state_locked(struct vcpu *vcpu, enum vcpu_state newstate)
{
	int error;

	if ((error = vcpu_set_state_locked(vcpu, newstate, false)) != 0)
		xhyve_abort("Error %d setting state to %d", error, newstate);
}

static void
vm_set_rendezvous_func(struct vm *vm, vm_rendezvous_func_t func)
{
	/*
	 * Update 'rendezvous_func' and execute a write memory barrier to
	 * ensure that it is visible across all host cpus. This is not needed
	 * for correctness but it does ensure that all the vcpus will notice
	 * that the rendezvous is requested immediately.
	 */
	vm->rendezvous_func = func;
	wmb();
}

#define	RENDEZVOUS_CTR0(vm, vcpuid, fmt) \
	do { \
		if (vcpuid >= 0) {\
			VCPU_CTR0(vm, vcpuid, fmt); \
		} else {\
			VM_CTR0(vm, fmt); \
		} \
	} while (0)

static void
vm_handle_rendezvous(struct vm *vm, int vcpuid)
{

	KASSERT(vcpuid == -1 || (vcpuid >= 0 && vcpuid < VM_MAXCPU),
	    ("vm_handle_rendezvous: invalid vcpuid %d", vcpuid));

	pthread_mutex_lock(&vm->rendezvous_mtx);
	while (vm->rendezvous_func != NULL) {
		/* 'rendezvous_req_cpus' must be a subset of 'active_cpus' */
		CPU_AND(&vm->rendezvous_req_cpus, &vm->active_cpus);

		if (vcpuid != -1 &&
		    CPU_ISSET(((unsigned) vcpuid), &vm->rendezvous_req_cpus) &&
		    !CPU_ISSET(((unsigned) vcpuid), &vm->rendezvous_done_cpus)) {
			VCPU_CTR0(vm, vcpuid, "Calling rendezvous func");
			(*vm->rendezvous_func)(vm, vcpuid, vm->rendezvous_arg);
			CPU_SET(((unsigned) vcpuid), &vm->rendezvous_done_cpus);
		}
		if (CPU_CMP(&vm->rendezvous_req_cpus,
		    &vm->rendezvous_done_cpus) == 0) {
			VCPU_CTR0(vm, vcpuid, "Rendezvous completed");
			vm_set_rendezvous_func(vm, NULL);
			pthread_cond_broadcast(&vm->rendezvous_sleep_cnd);
			//wakeup(&vm->rendezvous_func);
			break;
		}
		RENDEZVOUS_CTR0(vm, vcpuid, "Wait for rendezvous completion");
		pthread_cond_wait(&vm->rendezvous_sleep_cnd, &vm->rendezvous_mtx);
		//mtx_sleep(&vm->rendezvous_func, &vm->rendezvous_mtx, 0, "vmrndv", 0);
	}
	pthread_mutex_unlock(&vm->rendezvous_mtx);
}

/*
 * Emulate a guest 'hlt' by sleeping until the vcpu is ready to run.
 */
static int
vm_handle_hlt(struct vm *vm, int vcpuid, bool intr_disabled)
{
	struct vcpu *vcpu;
	const char *wmesg;
	int vcpu_halted, vm_halted;
	const struct timespec ts = {.tv_sec = 1, .tv_nsec = 0}; /* 1 second */

	KASSERT(!CPU_ISSET(((unsigned) vcpuid), &vm->halted_cpus),
		("vcpu already halted"));

	vcpu = &vm->vcpu[vcpuid];
	vcpu_halted = 0;
	vm_halted = 0;

	vcpu_lock(vcpu);
	while (1) {
		/*
		 * Do a final check for pending NMI or interrupts before
		 * really putting this thread to sleep. Also check for
		 * software events that would cause this vcpu to wakeup.
		 *
		 * These interrupts/events could have happened after the
		 * vcpu returned from VMRUN() and before it acquired the
		 * vcpu lock above.
		 */
		if (vm->rendezvous_func != NULL || vm->suspend)
			break;
		if (vm_nmi_pending(vm, vcpuid))
			break;
		if (!intr_disabled) {
			if (vm_extint_pending(vm, vcpuid) ||
			    vlapic_pending_intr(vcpu->vlapic, NULL)) {
				break;
			}
		}

		/*
		 * Some Linux guests implement "halt" by having all vcpus
		 * execute HLT with interrupts disabled. 'halted_cpus' keeps
		 * track of the vcpus that have entered this state. When all
		 * vcpus enter the halted state the virtual machine is halted.
		 */
		if (intr_disabled) {
			wmesg = "vmhalt";
			VCPU_CTR0(vm, vcpuid, "Halted");
			if (!vcpu_halted && halt_detection_enabled) {
				vcpu_halted = 1;
				CPU_SET_ATOMIC(((unsigned) vcpuid), &vm->halted_cpus);
			}
			if (CPU_CMP(&vm->halted_cpus, &vm->active_cpus) == 0) {
				vm_halted = 1;
				break;
			}
		} else {
			wmesg = "vmidle";
		}

		//t = ticks;
		vcpu_require_state_locked(vcpu, VCPU_SLEEPING);
		/*
		 * XXX msleep_spin() cannot be interrupted by signals so
		 * wake up periodically to check pending signals.
		 */
		pthread_mutex_lock(&vcpu->vcpu_sleep_mtx);
		vcpu_unlock(vcpu);
		pthread_cond_timedwait_relative_np(&vcpu->vcpu_sleep_cnd,
			&vcpu->vcpu_sleep_mtx, &ts);
		vcpu_lock(vcpu);
		pthread_mutex_unlock(&vcpu->vcpu_sleep_mtx);
		//msleep_spin(vcpu, &vcpu->mtx, wmesg, hz);
		vcpu_require_state_locked(vcpu, VCPU_FROZEN);
		//vmm_stat_incr(vm, vcpuid, VCPU_IDLE_TICKS, ticks - t);
	}

	if (vcpu_halted)
		CPU_CLR_ATOMIC(((unsigned) vcpuid), &vm->halted_cpus);

	vcpu_unlock(vcpu);

	if (vm_halted)
		vm_suspend(vm, VM_SUSPEND_HALT);

	return (0);
}

static int
vm_handle_inst_emul(struct vm *vm, int vcpuid, bool *retu)
{
	struct vie *vie;
	struct vcpu *vcpu;
	struct vm_exit *vme;
	uint64_t gla, gpa, cs_base;
	struct vm_guest_paging *paging;
	mem_region_read_t mread;
	mem_region_write_t mwrite;
	enum vm_cpu_mode cpu_mode;
	int cs_d, error, fault, length;

	vcpu = &vm->vcpu[vcpuid];
	vme = &vcpu->exitinfo;

	gla = vme->u.inst_emul.gla;
	gpa = vme->u.inst_emul.gpa;
	cs_base = vme->u.inst_emul.cs_base;
	cs_d = vme->u.inst_emul.cs_d;
	vie = &vme->u.inst_emul.vie;
	paging = &vme->u.inst_emul.paging;
	cpu_mode = paging->cpu_mode;

	VCPU_CTR1(vm, vcpuid, "inst_emul fault accessing gpa %#llx", gpa);

	/* Fetch, decode and emulate the faulting instruction */
	if (vie->num_valid == 0) {
		/*
		 * If the instruction length is not known then assume a
		 * maximum size instruction.
		 */
		length = vme->inst_length ? vme->inst_length : VIE_INST_SIZE;
		error = vmm_fetch_instruction(vm, vcpuid, paging, vme->rip +
		    cs_base, length, vie, &fault);
	} else {
		/*
		 * The instruction bytes have already been copied into 'vie'
		 */
		error = fault = 0;
	}
	if (error || fault)
		return (error);

	if (vmm_decode_instruction(vm, vcpuid, gla, cpu_mode, cs_d, vie) != 0) {
		VCPU_CTR1(vm, vcpuid, "Error decoding instruction at %#llx",
		    vme->rip + cs_base);
		*retu = true;	    /* dump instruction bytes in userspace */
		return (0);
	}

	/*
	 * If the instruction length was not specified then update it now
	 * along with 'nextrip'.
	 */
	if (vme->inst_length == 0) {
		vme->inst_length = vie->num_processed;
		vcpu->nextrip += vie->num_processed;
	}

	/* return to userland unless this is an in-kernel emulated device */
	if (gpa >= DEFAULT_APIC_BASE && gpa < DEFAULT_APIC_BASE + XHYVE_PAGE_SIZE) {
		mread = lapic_mmio_read;
		mwrite = lapic_mmio_write;
	} else if (gpa >= VIOAPIC_BASE && gpa < VIOAPIC_BASE + VIOAPIC_SIZE) {
		mread = vioapic_mmio_read;
		mwrite = vioapic_mmio_write;
	} else if (gpa >= VHPET_BASE && gpa < VHPET_BASE + VHPET_SIZE) {
		mread = vhpet_mmio_read;
		mwrite = vhpet_mmio_write;
	} else {
		*retu = true;
		return (0);
	}

	error = vmm_emulate_instruction(vm, vcpuid, gpa, vie, paging,
	    mread, mwrite, retu);

	return (error);
}

static int
vm_handle_suspend(struct vm *vm, int vcpuid, bool *retu)
{
	int i, done;
	struct vcpu *vcpu;
	const struct timespec ts = {.tv_sec = 1, .tv_nsec = 0}; /* 1 second */

	done = 0;
	vcpu = &vm->vcpu[vcpuid];

	CPU_SET_ATOMIC(((unsigned) vcpuid), &vm->suspended_cpus);

	/*
	 * Wait until all 'active_cpus' have suspended themselves.
	 *
	 * Since a VM may be suspended at any time including when one or
	 * more vcpus are doing a rendezvous we need to call the rendezvous
	 * handler while we are waiting to prevent a deadlock.
	 */
	vcpu_lock(vcpu);
	while (1) {
		if (CPU_CMP(&vm->suspended_cpus, &vm->active_cpus) == 0) {
			VCPU_CTR0(vm, vcpuid, "All vcpus suspended");
			break;
		}

		if (vm->rendezvous_func == NULL) {
			VCPU_CTR0(vm, vcpuid, "Sleeping during suspend");
			vcpu_require_state_locked(vcpu, VCPU_SLEEPING);

			pthread_mutex_lock(&vcpu->vcpu_sleep_mtx);
			vcpu_unlock(vcpu);
			pthread_cond_timedwait_relative_np(&vcpu->vcpu_sleep_cnd,
				&vcpu->vcpu_sleep_mtx, &ts);
			vcpu_lock(vcpu);
			pthread_mutex_unlock(&vcpu->vcpu_sleep_mtx);
			//msleep_spin(vcpu, &vcpu->mtx, "vmsusp", hz);

			vcpu_require_state_locked(vcpu, VCPU_FROZEN);
		} else {
			VCPU_CTR0(vm, vcpuid, "Rendezvous during suspend");
			vcpu_unlock(vcpu);
			vm_handle_rendezvous(vm, vcpuid);
			vcpu_lock(vcpu);
		}
	}
	vcpu_unlock(vcpu);

	/*
	 * Wakeup the other sleeping vcpus and return to userspace.
	 */
	for (i = 0; i < VM_MAXCPU; i++) {
		if (CPU_ISSET(((unsigned) i), &vm->suspended_cpus)) {
			vcpu_notify_event(vm, i, false);
		}
	}

	*retu = true;
	return (0);
}

int
vm_suspend(struct vm *vm, enum vm_suspend_how how)
{
	int i;

	if (how <= VM_SUSPEND_NONE || how >= VM_SUSPEND_LAST)
		return (EINVAL);

	if (atomic_cmpset_int(((volatile u_int *) &vm->suspend), 0, how) == 0) {
		VM_CTR2(vm, "virtual machine already suspended %d/%d",
		    vm->suspend, how);
		return (EALREADY);
	}

	VM_CTR1(vm, "virtual machine successfully suspended %d", how);

	/*
	 * Notify all active vcpus that they are now suspended.
	 */
	for (i = 0; i < VM_MAXCPU; i++) {
		if (CPU_ISSET(((unsigned) i), &vm->active_cpus))
			vcpu_notify_event(vm, i, false);
	}

	return (0);
}

void
vm_exit_suspended(struct vm *vm, int vcpuid, uint64_t rip)
{
	struct vm_exit *vmexit;

	KASSERT(vm->suspend > VM_SUSPEND_NONE && vm->suspend < VM_SUSPEND_LAST,
	    ("vm_exit_suspended: invalid suspend type %d", vm->suspend));

	vmexit = vm_exitinfo(vm, vcpuid);
	vmexit->rip = rip;
	vmexit->inst_length = 0;
	vmexit->exitcode = VM_EXITCODE_SUSPENDED;
	vmexit->u.suspended.how = (enum vm_suspend_how) vm->suspend;
}

void
vm_exit_rendezvous(struct vm *vm, int vcpuid, uint64_t rip)
{
	struct vm_exit *vmexit;

	KASSERT(vm->rendezvous_func != NULL, ("rendezvous not in progress"));

	vmexit = vm_exitinfo(vm, vcpuid);
	vmexit->rip = rip;
	vmexit->inst_length = 0;
	vmexit->exitcode = VM_EXITCODE_RENDEZVOUS;
	vmm_stat_incr(vm, vcpuid, VMEXIT_RENDEZVOUS, 1);
}

void
vm_vcpu_dump(struct vm *vm, int vcpuid)
{
	VCPU_DUMP(vm->cookie, vcpuid);
}

void pittest(struct vm *thevm);

int
vm_run(struct vm *vm, int vcpuid, struct vm_exit *vm_exit)
{
	int error;
	struct vcpu *vcpu;
	// uint64_t tscval;
	struct vm_exit *vme;
	bool retu, intr_disabled;
	void *rptr, *sptr;

	if (vcpuid < 0 || vcpuid >= VM_MAXCPU)
		return (EINVAL);

	if (!CPU_ISSET(((unsigned) vcpuid), &vm->active_cpus))
		return (EINVAL);

	if (CPU_ISSET(((unsigned) vcpuid), &vm->suspended_cpus))
		return (EINVAL);

	rptr = &vm->rendezvous_func;
	sptr = &vm->suspend;
	vcpu = &vm->vcpu[vcpuid];
	vme = &vcpu->exitinfo;
	retu = false;

restart:
	// tscval = rdtsc();

	vcpu_require_state(vm, vcpuid, VCPU_RUNNING);
	error = VMRUN(vm->cookie, vcpuid, (register_t) vcpu->nextrip, rptr, sptr);
	vcpu_require_state(vm, vcpuid, VCPU_FROZEN);


	// vmm_stat_incr(vm, vcpuid, VCPU_TOTAL_RUNTIME, rdtsc() - tscval);

	if (error == 0) {
		retu = false;
		vcpu->nextrip = vme->rip + ((unsigned) vme->inst_length);
		switch (((int) (vme->exitcode))) {
		case VM_EXITCODE_SUSPENDED:
			error = vm_handle_suspend(vm, vcpuid, &retu);
			break;
		case VM_EXITCODE_IOAPIC_EOI:
			vioapic_process_eoi(vm, vcpuid,
			    vme->u.ioapic_eoi.vector);
			break;
		case VM_EXITCODE_RENDEZVOUS:
			vm_handle_rendezvous(vm, vcpuid);
			error = 0;
			break;
		case VM_EXITCODE_HLT:
			intr_disabled = ((vme->u.hlt.rflags & PSL_I) == 0);
			error = vm_handle_hlt(vm, vcpuid, intr_disabled);
			break;
		case VM_EXITCODE_PAGING:
			error = 0;
			break;
		case VM_EXITCODE_INST_EMUL:
			error = vm_handle_inst_emul(vm, vcpuid, &retu);
			break;
		case VM_EXITCODE_INOUT:
		case VM_EXITCODE_INOUT_STR:
			error = vm_handle_inout(vm, vcpuid, vme, &retu);
			break;
		case VM_EXITCODE_MONITOR:
		case VM_EXITCODE_MWAIT:
			vm_inject_ud(vm, vcpuid);
			break;
		default:
			retu = true;	/* handled in userland */
			break;
		}
	}

	if (error == 0 && retu == false)
		goto restart;

	/* copy the exit information (FIXME: zero copy) */
	bcopy(vme, vm_exit, sizeof(struct vm_exit));
	return (error);
}

int
vm_restart_instruction(void *arg, int vcpuid)
{
	struct vm *vm;
	struct vcpu *vcpu;
	enum vcpu_state state;
	uint64_t rip;
	int error;

	vm = arg;
	if (vcpuid < 0 || vcpuid >= VM_MAXCPU)
		return (EINVAL);

	vcpu = &vm->vcpu[vcpuid];
	state = vcpu_get_state(vm, vcpuid);
	if (state == VCPU_RUNNING) {
		/*
		 * When a vcpu is "running" the next instruction is determined
		 * by adding 'rip' and 'inst_length' in the vcpu's 'exitinfo'.
		 * Thus setting 'inst_length' to zero will cause the current
		 * instruction to be restarted.
		 */
		vcpu->exitinfo.inst_length = 0;
		VCPU_CTR1(vm, vcpuid, "restarting instruction at %#llx by "
		    "setting inst_length to zero", vcpu->exitinfo.rip);
	} else if (state == VCPU_FROZEN) {
		/*
		 * When a vcpu is "frozen" it is outside the critical section
		 * around VMRUN() and 'nextrip' points to the next instruction.
		 * Thus instruction restart is achieved by setting 'nextrip'
		 * to the vcpu's %rip.
		 */
		error = vm_get_register(vm, vcpuid, VM_REG_GUEST_RIP, &rip);
		KASSERT(!error, ("%s: error %d getting rip", __func__, error));
		VCPU_CTR2(vm, vcpuid, "restarting instruction by updating "
		    "nextrip from %#llx to %#llx", vcpu->nextrip, rip);
		vcpu->nextrip = rip;
	} else {
		xhyve_abort("%s: invalid state %d\n", __func__, state);
	}
	return (0);
}

int
vm_exit_intinfo(struct vm *vm, int vcpuid, uint64_t info)
{
	struct vcpu *vcpu;
	int type, vector;

	if (vcpuid < 0 || vcpuid >= VM_MAXCPU)
		return (EINVAL);

	vcpu = &vm->vcpu[vcpuid];

	if (info & VM_INTINFO_VALID) {
		type = info & VM_INTINFO_TYPE;
		vector = info & 0xff;
		if (type == VM_INTINFO_NMI && vector != IDT_NMI)
			return (EINVAL);
		if (type == VM_INTINFO_HWEXCEPTION && vector >= 32)
			return (EINVAL);
		if (info & VM_INTINFO_RSVD)
			return (EINVAL);
	} else {
		info = 0;
	}
	VCPU_CTR2(vm, vcpuid, "%s: info1(%#llx)", __func__, info);
	vcpu->exitintinfo = info;
	return (0);
}

enum exc_class {
	EXC_BENIGN,
	EXC_CONTRIBUTORY,
	EXC_PAGEFAULT
};

#define	IDT_VE	20	/* Virtualization Exception (Intel specific) */

static enum exc_class
exception_class(uint64_t info)
{
	int type, vector;

	KASSERT(info & VM_INTINFO_VALID, ("intinfo must be valid: %#llx", info));
	type = info & VM_INTINFO_TYPE;
	vector = info & 0xff;

	/* Table 6-4, "Interrupt and Exception Classes", Intel SDM, Vol 3 */
	switch (type) {
	case VM_INTINFO_HWINTR:
	case VM_INTINFO_SWINTR:
	case VM_INTINFO_NMI:
		return (EXC_BENIGN);
	default:
		/*
		 * Hardware exception.
		 *
		 * SVM and VT-x use identical type values to represent NMI,
		 * hardware interrupt and software interrupt.
		 *
		 * SVM uses type '3' for all exceptions. VT-x uses type '3'
		 * for exceptions except #BP and #OF. #BP and #OF use a type
		 * value of '5' or '6'. Therefore we don't check for explicit
		 * values of 'type' to classify 'intinfo' into a hardware
		 * exception.
		 */
		break;
	}

	switch (vector) {
	case IDT_PF:
	case IDT_VE:
		return (EXC_PAGEFAULT);
	case IDT_DE:
	case IDT_TS:
	case IDT_NP:
	case IDT_SS:
	case IDT_GP:
		return (EXC_CONTRIBUTORY);
	default:
		return (EXC_BENIGN);
	}
}

static int
nested_fault(struct vm *vm, int vcpuid, uint64_t info1, uint64_t info2,
    uint64_t *retinfo)
{
	enum exc_class exc1, exc2;
	int type1, vector1;

	KASSERT(info1 & VM_INTINFO_VALID, ("info1 %#llx is not valid", info1));
	KASSERT(info2 & VM_INTINFO_VALID, ("info2 %#llx is not valid", info2));

	/*
	 * If an exception occurs while attempting to call the double-fault
	 * handler the processor enters shutdown mode (aka triple fault).
	 */
	type1 = info1 & VM_INTINFO_TYPE;
	vector1 = info1 & 0xff;
	if (type1 == VM_INTINFO_HWEXCEPTION && vector1 == IDT_DF) {
		VCPU_CTR2(vm, vcpuid, "triple fault: info1(%#llx), info2(%#llx)",
		    info1, info2);
		vm_suspend(vm, VM_SUSPEND_TRIPLEFAULT);
		*retinfo = 0;
		return (0);
	}

	/*
	 * Table 6-5 "Conditions for Generating a Double Fault", Intel SDM, Vol3
	 */
	exc1 = exception_class(info1);
	exc2 = exception_class(info2);
	if ((exc1 == EXC_CONTRIBUTORY && exc2 == EXC_CONTRIBUTORY) ||
	    (exc1 == EXC_PAGEFAULT && exc2 != EXC_BENIGN)) {
		/* Convert nested fault into a double fault. */
		*retinfo = IDT_DF;
		*retinfo |= VM_INTINFO_VALID | VM_INTINFO_HWEXCEPTION;
		*retinfo |= VM_INTINFO_DEL_ERRCODE;
	} else {
		/* Handle exceptions serially */
		*retinfo = info2;
	}
	return (1);
}

static uint64_t
vcpu_exception_intinfo(struct vcpu *vcpu)
{
	uint64_t info = 0;

	if (vcpu->exception_pending) {
		info = vcpu->exc_vector & 0xff;
		info |= VM_INTINFO_VALID | VM_INTINFO_HWEXCEPTION;
		if (vcpu->exc_errcode_valid) {
			info |= VM_INTINFO_DEL_ERRCODE;
			info |= (uint64_t)vcpu->exc_errcode << 32;
		}
	}
	return (info);
}

int
vm_entry_intinfo(struct vm *vm, int vcpuid, uint64_t *retinfo)
{
	struct vcpu *vcpu;
	uint64_t info1, info2;
	int valid;

	KASSERT(vcpuid >= 0 && vcpuid < VM_MAXCPU, ("invalid vcpu %d", vcpuid));

	vcpu = &vm->vcpu[vcpuid];

	info1 = vcpu->exitintinfo;
	vcpu->exitintinfo = 0;

	info2 = 0;
	if (vcpu->exception_pending) {
		info2 = vcpu_exception_intinfo(vcpu);
		vcpu->exception_pending = 0;
		VCPU_CTR2(vm, vcpuid, "Exception %d delivered: %#llx",
		    vcpu->exc_vector, info2);
	}

	if ((info1 & VM_INTINFO_VALID) && (info2 & VM_INTINFO_VALID)) {
		valid = nested_fault(vm, vcpuid, info1, info2, retinfo);
	} else if (info1 & VM_INTINFO_VALID) {
		*retinfo = info1;
		valid = 1;
	} else if (info2 & VM_INTINFO_VALID) {
		*retinfo = info2;
		valid = 1;
	} else {
		valid = 0;
	}

	if (valid) {
		VCPU_CTR4(vm, vcpuid, "%s: info1(%#llx), info2(%#llx), "
		    "retinfo(%#llx)", __func__, info1, info2, *retinfo);
	}

	return (valid);
}

int
vm_get_intinfo(struct vm *vm, int vcpuid, uint64_t *info1, uint64_t *info2)
{
	struct vcpu *vcpu;

	if (vcpuid < 0 || vcpuid >= VM_MAXCPU)
		return (EINVAL);

	vcpu = &vm->vcpu[vcpuid];
	*info1 = vcpu->exitintinfo;
	*info2 = vcpu_exception_intinfo(vcpu);
	return (0);
}

int
vm_inject_exception(struct vm *vm, int vcpuid, int vector, int errcode_valid,
    uint32_t errcode, int restart_instruction)
{
	struct vcpu *vcpu;
	int error;

	if (vcpuid < 0 || vcpuid >= VM_MAXCPU)
		return (EINVAL);

	if (vector < 0 || vector >= 32)
		return (EINVAL);

	/*
	 * A double fault exception should never be injected directly into
	 * the guest. It is a derived exception that results from specific
	 * combinations of nested faults.
	 */
	if (vector == IDT_DF)
		return (EINVAL);

	vcpu = &vm->vcpu[vcpuid];

	if (vcpu->exception_pending) {
		VCPU_CTR2(vm, vcpuid, "Unable to inject exception %d due to "
		    "pending exception %d", vector, vcpu->exc_vector);
		return (EBUSY);
	}

	/*
	 * From section 26.6.1 "Interruptibility State" in Intel SDM:
	 *
	 * Event blocking by "STI" or "MOV SS" is cleared after guest executes
	 * one instruction or incurs an exception.
	 */
	error = vm_set_register(vm, vcpuid, VM_REG_GUEST_INTR_SHADOW, 0);
	KASSERT(error == 0, ("%s: error %d clearing interrupt shadow",
	    __func__, error));

	if (restart_instruction)
		vm_restart_instruction(vm, vcpuid);

	vcpu->exception_pending = 1;
	vcpu->exc_vector = vector;
	vcpu->exc_errcode = errcode;
	vcpu->exc_errcode_valid = errcode_valid;
	VCPU_CTR1(vm, vcpuid, "Exception %d pending", vector);
	return (0);
}

void
vm_inject_fault(void *vmarg, int vcpuid, int vector, int errcode_valid,
    int errcode)
{
	struct vm *vm;
	int error, restart_instruction;

	vm = vmarg;
	restart_instruction = 1;

	error = vm_inject_exception(vm, vcpuid, vector, errcode_valid,
	    ((uint32_t) errcode), restart_instruction);
	KASSERT(error == 0, ("vm_inject_exception error %d", error));
}

void
vm_inject_pf(void *vmarg, int vcpuid, int error_code, uint64_t cr2)
{
	struct vm *vm;
	int error;

	vm = vmarg;
	VCPU_CTR2(vm, vcpuid, "Injecting page fault: error_code %#x, cr2 %#llx",
	    error_code, cr2);

	error = vm_set_register(vm, vcpuid, VM_REG_GUEST_CR2, cr2);
	KASSERT(error == 0, ("vm_set_register(cr2) error %d", error));

	vm_inject_fault(vm, vcpuid, IDT_PF, 1, error_code);
}

static VMM_STAT(VCPU_NMI_COUNT, "number of NMIs delivered to vcpu");

int
vm_inject_nmi(struct vm *vm, int vcpuid)
{
	struct vcpu *vcpu;

	if (vcpuid < 0 || vcpuid >= VM_MAXCPU)
		return (EINVAL);

	vcpu = &vm->vcpu[vcpuid];

	vcpu->nmi_pending = 1;
	vcpu_notify_event(vm, vcpuid, false);
	return (0);
}

int
vm_nmi_pending(struct vm *vm, int vcpuid)
{
	struct vcpu *vcpu;

	if (vcpuid < 0 || vcpuid >= VM_MAXCPU)
		xhyve_abort("vm_nmi_pending: invalid vcpuid %d\n", vcpuid);

	vcpu = &vm->vcpu[vcpuid];

	return (vcpu->nmi_pending);
}

void
vm_nmi_clear(struct vm *vm, int vcpuid)
{
	struct vcpu *vcpu;

	if (vcpuid < 0 || vcpuid >= VM_MAXCPU)
		xhyve_abort("vm_nmi_pending: invalid vcpuid %d\n", vcpuid);

	vcpu = &vm->vcpu[vcpuid];

	if (vcpu->nmi_pending == 0)
		xhyve_abort("vm_nmi_clear: inconsistent nmi_pending state\n");

	vcpu->nmi_pending = 0;
	vmm_stat_incr(vm, vcpuid, VCPU_NMI_COUNT, 1);
}

static VMM_STAT(VCPU_EXTINT_COUNT, "number of ExtINTs delivered to vcpu");

int
vm_inject_extint(struct vm *vm, int vcpuid)
{
	struct vcpu *vcpu;

	if (vcpuid < 0 || vcpuid >= VM_MAXCPU)
		return (EINVAL);

	vcpu = &vm->vcpu[vcpuid];

	vcpu->extint_pending = 1;
	vcpu_notify_event(vm, vcpuid, false);
	return (0);
}

int
vm_extint_pending(struct vm *vm, int vcpuid)
{
	struct vcpu *vcpu;

	if (vcpuid < 0 || vcpuid >= VM_MAXCPU)
		xhyve_abort("vm_extint_pending: invalid vcpuid %d\n", vcpuid);

	vcpu = &vm->vcpu[vcpuid];

	return (vcpu->extint_pending);
}

void
vm_extint_clear(struct vm *vm, int vcpuid)
{
	struct vcpu *vcpu;

	if (vcpuid < 0 || vcpuid >= VM_MAXCPU)
		xhyve_abort("vm_extint_pending: invalid vcpuid %d\n", vcpuid);

	vcpu = &vm->vcpu[vcpuid];

	if (vcpu->extint_pending == 0)
		xhyve_abort("vm_extint_clear: inconsistent extint_pending state\n");

	vcpu->extint_pending = 0;
	vmm_stat_incr(vm, vcpuid, VCPU_EXTINT_COUNT, 1);
}

int
vm_get_capability(struct vm *vm, int vcpu, int type, int *retval)
{
	if (vcpu < 0 || vcpu >= VM_MAXCPU)
		return (EINVAL);

	if (type < 0 || type >= VM_CAP_MAX)
		return (EINVAL);

	return (VMGETCAP(vm->cookie, vcpu, type, retval));
}

int
vm_set_capability(struct vm *vm, int vcpu, int type, int val)
{
	if (vcpu < 0 || vcpu >= VM_MAXCPU)
		return (EINVAL);

	if (type < 0 || type >= VM_CAP_MAX)
		return (EINVAL);

	return (VMSETCAP(vm->cookie, vcpu, type, val));
}

struct vlapic *
vm_lapic(struct vm *vm, int cpu)
{
	return (vm->vcpu[cpu].vlapic);
}

struct vioapic *
vm_ioapic(struct vm *vm)
{

	return (vm->vioapic);
}

struct vhpet *
vm_hpet(struct vm *vm)
{

	return (vm->vhpet);
}

int
vcpu_set_state(struct vm *vm, int vcpuid, enum vcpu_state newstate,
    bool from_idle)
{
	int error;
	struct vcpu *vcpu;

	if (vcpuid < 0 || vcpuid >= VM_MAXCPU)
		xhyve_abort("vm_set_run_state: invalid vcpuid %d\n", vcpuid);

	vcpu = &vm->vcpu[vcpuid];

	vcpu_lock(vcpu);
	error = vcpu_set_state_locked(vcpu, newstate, from_idle);
	vcpu_unlock(vcpu);
	return (error);
}

enum vcpu_state
vcpu_get_state(struct vm *vm, int vcpuid)
{
	struct vcpu *vcpu;
	enum vcpu_state state;

	if (vcpuid < 0 || vcpuid >= VM_MAXCPU)
		xhyve_abort("vm_get_run_state: invalid vcpuid %d\n", vcpuid);

	vcpu = &vm->vcpu[vcpuid];

	vcpu_lock(vcpu);
	state = vcpu->state;
	vcpu_unlock(vcpu);

	return (state);
}

int
vm_activate_cpu(struct vm *vm, int vcpuid)
{

	if (vcpuid < 0 || vcpuid >= VM_MAXCPU)
		return (EINVAL);

	if (CPU_ISSET(((unsigned) vcpuid), &vm->active_cpus))
		return (EBUSY);

	VCPU_CTR0(vm, vcpuid, "activated");
	CPU_SET_ATOMIC(((unsigned) vcpuid), &vm->active_cpus);
	return (0);
}

cpuset_t
vm_active_cpus(struct vm *vm)
{

	return (vm->active_cpus);
}

cpuset_t
vm_suspended_cpus(struct vm *vm)
{

	return (vm->suspended_cpus);
}

void *
vcpu_stats(struct vm *vm, int vcpuid)
{

	return (vm->vcpu[vcpuid].stats);
}

int
vm_get_x2apic_state(struct vm *vm, int vcpuid, enum x2apic_state *state)
{
	if (vcpuid < 0 || vcpuid >= VM_MAXCPU)
		return (EINVAL);

	*state = vm->vcpu[vcpuid].x2apic_state;

	return (0);
}

int
vm_set_x2apic_state(struct vm *vm, int vcpuid, enum x2apic_state state)
{
	if (vcpuid < 0 || vcpuid >= VM_MAXCPU)
		return (EINVAL);

	if (state >= X2APIC_STATE_LAST)
		return (EINVAL);

	vm->vcpu[vcpuid].x2apic_state = state;

	vlapic_set_x2apic_state(vm, vcpuid, state);

	return (0);
}

/*
 * This function is called to ensure that a vcpu "sees" a pending event
 * as soon as possible:
 * - If the vcpu thread is sleeping then it is woken up.
 * - If the vcpu is running on a different host_cpu then an IPI will be directed
 *   to the host_cpu to cause the vcpu to trap into the hypervisor.
 */
void
vcpu_notify_event(struct vm *vm, int vcpuid, UNUSED bool lapic_intr)
{
	struct vcpu *vcpu;

	vcpu = &vm->vcpu[vcpuid];
	vcpu_lock(vcpu);
	if (vcpu->state == VCPU_RUNNING) {
		VCPU_INTERRUPT(vcpuid);
		/* FIXME */
		// if (hostcpu != curcpu) {
		// 	if (lapic_intr) {
		// 		vlapic_post_intr(vcpu->vlapic, hostcpu,
		// 		    vmm_ipinum);
		// 	} else {
		// 		ipi_cpu(hostcpu, vmm_ipinum);
		// 	}
		// } else {
		//	/*
		//	 * If the 'vcpu' is running on 'curcpu' then it must
		//	 * be sending a notification to itself (e.g. SELF_IPI).
		//	 * The pending event will be picked up when the vcpu
		//	 * transitions back to guest context.
		//	 */
		// }
	} else {
		if (vcpu->state == VCPU_SLEEPING)
			pthread_cond_signal(&vcpu->vcpu_sleep_cnd);
			//wakeup_one(vcpu);
	}
	vcpu_unlock(vcpu);
}

int
vm_apicid2vcpuid(UNUSED struct vm *vm, int apicid)
{
	/*
	 * XXX apic id is assumed to be numerically identical to vcpu id
	 */
	return (apicid);
}

void
vm_smp_rendezvous(struct vm *vm, int vcpuid, cpuset_t dest,
    vm_rendezvous_func_t func, void *arg)
{
	int i;

	KASSERT(vcpuid == -1 || (vcpuid >= 0 && vcpuid < VM_MAXCPU),
	    ("vm_smp_rendezvous: invalid vcpuid %d", vcpuid));

restart:
	pthread_mutex_lock(&vm->rendezvous_mtx);
	if (vm->rendezvous_func != NULL) {
		/*
		 * If a rendezvous is already in progress then we need to
		 * call the rendezvous handler in case this 'vcpuid' is one
		 * of the targets of the rendezvous.
		 */
		RENDEZVOUS_CTR0(vm, vcpuid, "Rendezvous already in progress");
		pthread_mutex_unlock(&vm->rendezvous_mtx);
		vm_handle_rendezvous(vm, vcpuid);
		goto restart;
	}
	KASSERT(vm->rendezvous_func == NULL, ("vm_smp_rendezvous: previous "
	    "rendezvous is still in progress"));

	RENDEZVOUS_CTR0(vm, vcpuid, "Initiating rendezvous");
	vm->rendezvous_req_cpus = dest;
	CPU_ZERO(&vm->rendezvous_done_cpus);
	vm->rendezvous_arg = arg;
	vm_set_rendezvous_func(vm, func);
	pthread_mutex_unlock(&vm->rendezvous_mtx);

	/*
	 * Wake up any sleeping vcpus and trigger a VM-exit in any running
	 * vcpus so they handle the rendezvous as soon as possible.
	 */
	for (i = 0; i < VM_MAXCPU; i++) {
		if (CPU_ISSET(((unsigned) i), &dest))
			vcpu_notify_event(vm, i, false);
	}

	vm_handle_rendezvous(vm, vcpuid);
}

struct vatpic *
vm_atpic(struct vm *vm)
{
	return (vm->vatpic);
}

struct vatpit *
vm_atpit(struct vm *vm)
{
	return (vm->vatpit);
}

struct vpmtmr *
vm_pmtmr(struct vm *vm)
{

	return (vm->vpmtmr);
}

struct vrtc *
vm_rtc(struct vm *vm)
{

	return (vm->vrtc);
}

enum vm_reg_name
vm_segment_name(int seg)
{
	static enum vm_reg_name seg_names[] = {
		VM_REG_GUEST_ES,
		VM_REG_GUEST_CS,
		VM_REG_GUEST_SS,
		VM_REG_GUEST_DS,
		VM_REG_GUEST_FS,
		VM_REG_GUEST_GS
	};

	KASSERT(seg >= 0 && seg < ((int) nitems(seg_names)),
	    ("%s: invalid segment encoding %d", __func__, seg));
	return (seg_names[seg]);
}

void
vm_copy_teardown(UNUSED struct vm *vm, UNUSED int vcpuid,
	struct vm_copyinfo *copyinfo, int num_copyinfo)
{
	bzero(copyinfo, ((unsigned) num_copyinfo) * sizeof(struct vm_copyinfo));
}

int
vm_copy_setup(struct vm *vm, int vcpuid, struct vm_guest_paging *paging,
    uint64_t gla, size_t len, int prot, struct vm_copyinfo *copyinfo,
    int num_copyinfo, int *fault)
{
	int error, idx, nused;
	size_t n, off, remaining;
	void *hva;
	uint64_t gpa;

	bzero(copyinfo, sizeof(struct vm_copyinfo) * ((unsigned) num_copyinfo));

	nused = 0;
	remaining = len;
	while (remaining > 0) {
		KASSERT(nused < num_copyinfo, ("insufficient vm_copyinfo"));
		error = vm_gla2gpa(vm, vcpuid, paging, gla, prot, &gpa, fault);
		if (error || *fault)
			return (error);
		off = gpa & XHYVE_PAGE_MASK;
		n = min(remaining, XHYVE_PAGE_SIZE - off);
		copyinfo[nused].gpa = gpa;
		copyinfo[nused].len = n;
		remaining -= n;
		gla += n;
		nused++;
	}

	for (idx = 0; idx < nused; idx++) {
		hva = vm_gpa2hva(vm, copyinfo[idx].gpa, copyinfo[idx].len);
		if (hva == NULL)
			break;
		copyinfo[idx].hva = hva;
	}

	if (idx != nused) {
		vm_copy_teardown(vm, vcpuid, copyinfo, num_copyinfo);
		return (EFAULT);
	} else {
		*fault = 0;
		return (0);
	}
}

void
vm_copyin(UNUSED struct vm *vm, UNUSED int vcpuid, struct vm_copyinfo *copyinfo,
	void *kaddr, size_t len)
{
	char *dst;
	int idx;

	dst = kaddr;
	idx = 0;
	while (len > 0) {
		bcopy(copyinfo[idx].hva, dst, copyinfo[idx].len);
		len -= copyinfo[idx].len;
		dst += copyinfo[idx].len;
		idx++;
	}
}

void
vm_copyout(UNUSED struct vm *vm, UNUSED int vcpuid, const void *kaddr,
    struct vm_copyinfo *copyinfo, size_t len)
{
	const char *src;
	int idx;

	src = kaddr;
	idx = 0;
	while (len > 0) {
		bcopy(src, copyinfo[idx].hva, copyinfo[idx].len);
		len -= copyinfo[idx].len;
		src += copyinfo[idx].len;
		idx++;
	}
}
