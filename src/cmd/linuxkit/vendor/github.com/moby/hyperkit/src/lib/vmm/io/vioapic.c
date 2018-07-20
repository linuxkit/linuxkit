/*-
 * Copyright (c) 2013 Tycho Nightingale <tycho.nightingale@pluribusnetworks.com>
 * Copyright (c) 2013 Neel Natu <neel@freebsd.org>
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
#include <errno.h>
#include <assert.h>
#include <xhyve/support/misc.h>
#include <xhyve/support/apicreg.h>
#include <xhyve/vmm/vmm_ktr.h>
#include <xhyve/vmm/io/vioapic.h>
#include <xhyve/vmm/io/vlapic.h>

#define	IOREGSEL	0x00
#define	IOWIN		0x10

#define	REDIR_ENTRIES	24
#define	RTBL_RO_BITS	((uint64_t)(IOART_REM_IRR | IOART_DELIVS))

struct vioapic {
	struct vm *vm;
	pthread_mutex_t lock;
	uint32_t id;
	uint32_t ioregsel;
	struct {
		uint64_t reg;
		int acnt; /* sum of pin asserts (+1) and deasserts (-1) */
	} rtbl[REDIR_ENTRIES];
};

#define VIOAPIC_LOCK_INIT(v) xpthread_mutex_init(&(v)->lock);
#define VIOAPIC_LOCK(v) xpthread_mutex_lock(&(v)->lock)
#define VIOAPIC_UNLOCK(v) xpthread_mutex_unlock(&(v)->lock)

#define	VIOAPIC_CTR1(vioapic, fmt, a1) \
	VM_CTR1((vioapic)->vm, fmt, a1)

#define	VIOAPIC_CTR2(vioapic, fmt, a1, a2) \
	VM_CTR2((vioapic)->vm, fmt, a1, a2)

#ifdef XHYVE_CONFIG_TRACE
#define	VIOAPIC_CTR3(vioapic, fmt, a1, a2, a3) \
	VM_CTR3((vioapic)->vm, fmt, a1, a2, a3)
#endif

#ifdef XHYVE_CONFIG_TRACE
static const char *
pinstate_str(bool asserted)
{

	if (asserted)
		return ("asserted");
	else
		return ("deasserted");
}
#endif

static void
vioapic_send_intr(struct vioapic *vioapic, int pin)
{
	int vector, delmode;
	uint32_t low, high, dest;
	bool level, phys;

	KASSERT(pin >= 0 && pin < REDIR_ENTRIES,
	    ("vioapic_set_pinstate: invalid pin number %d", pin));

	low = (uint32_t) vioapic->rtbl[pin].reg;
	high = (uint32_t) (vioapic->rtbl[pin].reg >> 32);

	if ((low & IOART_INTMASK) == IOART_INTMSET) {
		VIOAPIC_CTR1(vioapic, "ioapic pin%d: masked", pin);
		return;
	}

	phys = ((low & IOART_DESTMOD) == IOART_DESTPHY);
	delmode = low & IOART_DELMOD;
	level = low & IOART_TRGRLVL ? true : false;
	if (level)
		vioapic->rtbl[pin].reg |= IOART_REM_IRR;

	vector = low & IOART_INTVEC;
	dest = high >> APIC_ID_SHIFT;
	vlapic_deliver_intr(vioapic->vm, level, dest, phys, delmode, vector);
}

static void
vioapic_set_pinstate(struct vioapic *vioapic, int pin, bool newstate)
{
	int oldcnt, newcnt;
	bool needintr;

	KASSERT(pin >= 0 && pin < REDIR_ENTRIES,
	    ("vioapic_set_pinstate: invalid pin number %d", pin));

	oldcnt = vioapic->rtbl[pin].acnt;
	if (newstate)
		vioapic->rtbl[pin].acnt++;
	else
		vioapic->rtbl[pin].acnt--;
	newcnt = vioapic->rtbl[pin].acnt;

	if (newcnt < 0) {
		VIOAPIC_CTR2(vioapic, "ioapic pin%d: bad acnt %d",
		    pin, newcnt);
	}

	needintr = false;
	if (oldcnt == 0 && newcnt == 1) {
		needintr = true;
		VIOAPIC_CTR1(vioapic, "ioapic pin%d: asserted", pin);
	} else if (oldcnt == 1 && newcnt == 0) {
		VIOAPIC_CTR1(vioapic, "ioapic pin%d: deasserted", pin);
	} else {
#ifdef XHYVE_CONFIG_TRACE
		VIOAPIC_CTR3(vioapic, "ioapic pin%d: %s, ignored, acnt %d",
		    pin, pinstate_str(newstate), newcnt);
#endif
	}
	if (needintr)
		vioapic_send_intr(vioapic, pin);
}

enum irqstate {
	IRQSTATE_ASSERT,
	IRQSTATE_DEASSERT,
	IRQSTATE_PULSE
};

static int
vioapic_set_irqstate(struct vm *vm, int irq, enum irqstate irqstate)
{
	struct vioapic *vioapic;

	if (irq < 0 || irq >= REDIR_ENTRIES)
		return (EINVAL);

	vioapic = vm_ioapic(vm);

	VIOAPIC_LOCK(vioapic);
	switch (irqstate) {
	case IRQSTATE_ASSERT:
		vioapic_set_pinstate(vioapic, irq, true);
		break;
	case IRQSTATE_DEASSERT:
		vioapic_set_pinstate(vioapic, irq, false);
		break;
	case IRQSTATE_PULSE:
		vioapic_set_pinstate(vioapic, irq, true);
		vioapic_set_pinstate(vioapic, irq, false);
		break;
	}
	VIOAPIC_UNLOCK(vioapic);

	return (0);
}

int
vioapic_assert_irq(struct vm *vm, int irq)
{

	return (vioapic_set_irqstate(vm, irq, IRQSTATE_ASSERT));
}

int
vioapic_deassert_irq(struct vm *vm, int irq)
{

	return (vioapic_set_irqstate(vm, irq, IRQSTATE_DEASSERT));
}

int
vioapic_pulse_irq(struct vm *vm, int irq)
{

	return (vioapic_set_irqstate(vm, irq, IRQSTATE_PULSE));
}

/*
 * Reset the vlapic's trigger-mode register to reflect the ioapic pin
 * configuration.
 */
static void
vioapic_update_tmr(struct vm *vm, int vcpuid, UNUSED void *arg)
{
	struct vioapic *vioapic;
	struct vlapic *vlapic;
	uint32_t low, high, dest;
	int delmode, pin, vector;
	bool level, phys;

	vlapic = vm_lapic(vm, vcpuid);
	vioapic = vm_ioapic(vm);

	VIOAPIC_LOCK(vioapic);
	/*
	 * Reset all vectors to be edge-triggered.
	 */
	vlapic_reset_tmr(vlapic);
	for (pin = 0; pin < REDIR_ENTRIES; pin++) {
		low = (uint32_t) vioapic->rtbl[pin].reg;
		high = (uint32_t) (vioapic->rtbl[pin].reg >> 32);

		level = low & IOART_TRGRLVL ? true : false;
		if (!level)
			continue;

		/*
		 * For a level-triggered 'pin' let the vlapic figure out if
		 * an assertion on this 'pin' would result in an interrupt
		 * being delivered to it. If yes, then it will modify the
		 * TMR bit associated with this vector to level-triggered.
		 */
		phys = ((low & IOART_DESTMOD) == IOART_DESTPHY);
		delmode = low & IOART_DELMOD;
		vector = low & IOART_INTVEC;
		dest = high >> APIC_ID_SHIFT;
		vlapic_set_tmr_level(vlapic, dest, phys, delmode, vector);
	}
	VIOAPIC_UNLOCK(vioapic);
}

static uint32_t
vioapic_read(struct vioapic *vioapic, UNUSED int vcpuid, uint32_t addr)
{
	int regnum, pin, rshift;

	regnum = addr & 0xff;
	switch (regnum) {
	case IOAPIC_ID:
		return (vioapic->id);
	case IOAPIC_VER:
		return (((REDIR_ENTRIES - 1) << MAXREDIRSHIFT) | 0x11);
	case IOAPIC_ARB:
		return (vioapic->id);
	default:
		break;
	}

	/* redirection table entries */
	if (regnum >= IOAPIC_REDTBL &&
	    regnum < IOAPIC_REDTBL + REDIR_ENTRIES * 2) {
		pin = (regnum - IOAPIC_REDTBL) / 2;
		if ((regnum - IOAPIC_REDTBL) % 2)
			rshift = 32;
		else
			rshift = 0;

		return ((uint32_t) (vioapic->rtbl[pin].reg >> rshift));
	}

	return (0);
}

static void
vioapic_write(struct vioapic *vioapic, int vcpuid, uint32_t addr, uint32_t data)
{
	uint64_t data64, mask64;
	uint64_t last, changed;
	int regnum, pin, lshift;
	cpuset_t allvcpus;

	regnum = addr & 0xff;
	switch (regnum) {
	case IOAPIC_ID:
		vioapic->id = data & APIC_ID_MASK;
		break;
	case IOAPIC_VER:
	case IOAPIC_ARB:
		/* readonly */
		break;
	default:
		break;
	}

	/* redirection table entries */
	if (regnum >= IOAPIC_REDTBL &&
	    regnum < IOAPIC_REDTBL + REDIR_ENTRIES * 2) {
		pin = (regnum - IOAPIC_REDTBL) / 2;
		if ((regnum - IOAPIC_REDTBL) % 2)
			lshift = 32;
		else
			lshift = 0;

		last = vioapic->rtbl[pin].reg;

		data64 = (uint64_t)data << lshift;
		mask64 = (uint64_t)0xffffffff << lshift;
		vioapic->rtbl[pin].reg &= ~mask64 | RTBL_RO_BITS;
		vioapic->rtbl[pin].reg |= data64 & ~RTBL_RO_BITS;

		VIOAPIC_CTR2(vioapic, "ioapic pin%d: redir table entry %#llx",
		    pin, vioapic->rtbl[pin].reg);

		/*
		 * If any fields in the redirection table entry (except mask
		 * or polarity) have changed then rendezvous all the vcpus
		 * to update their vlapic trigger-mode registers.
		 */
		changed = last ^ vioapic->rtbl[pin].reg;
		if (changed & ~((uint64_t) (IOART_INTMASK | IOART_INTPOL))) {
			VIOAPIC_CTR1(vioapic, "ioapic pin%d: recalculate "
			    "vlapic trigger-mode register", pin);
			VIOAPIC_UNLOCK(vioapic);
			allvcpus = vm_active_cpus(vioapic->vm);
			vm_smp_rendezvous(vioapic->vm, vcpuid, allvcpus,
			    vioapic_update_tmr, NULL);
			VIOAPIC_LOCK(vioapic);
		}

		/*
		 * Generate an interrupt if the following conditions are met:
		 * - pin is not masked
		 * - previous interrupt has been EOIed
		 * - pin level is asserted
		 */
		if ((vioapic->rtbl[pin].reg & IOART_INTMASK) == IOART_INTMCLR &&
		    (vioapic->rtbl[pin].reg & IOART_REM_IRR) == 0 &&
		    (vioapic->rtbl[pin].acnt > 0)) {
			VIOAPIC_CTR2(vioapic, "ioapic pin%d: asserted at rtbl "
			    "write, acnt %d", pin, vioapic->rtbl[pin].acnt);
			vioapic_send_intr(vioapic, pin);
		}
	}
}

static int
vioapic_mmio_rw(struct vioapic *vioapic, int vcpuid, uint64_t gpa,
    uint64_t *data, int size, bool doread)
{
	uint64_t offset;

	offset = gpa - VIOAPIC_BASE;

	/*
	 * The IOAPIC specification allows 32-bit wide accesses to the
	 * IOREGSEL (offset 0) and IOWIN (offset 16) registers.
	 */
	if (size != 4 || (offset != IOREGSEL && offset != IOWIN)) {
		if (doread)
			*data = 0;
		return (0);
	}

	VIOAPIC_LOCK(vioapic);
	if (offset == IOREGSEL) {
		if (doread)
			*data = vioapic->ioregsel;
		else
			vioapic->ioregsel = (uint32_t) *data;
	} else {
		if (doread) {
			*data = vioapic_read(vioapic, vcpuid,
			    vioapic->ioregsel);
		} else {
			vioapic_write(vioapic, vcpuid, vioapic->ioregsel,
				((uint32_t) *data));
		}
	}
	VIOAPIC_UNLOCK(vioapic);

	return (0);
}

int
vioapic_mmio_read(void *vm, int vcpuid, uint64_t gpa, uint64_t *rval,
    int size, UNUSED void *arg)
{
	int error;
	struct vioapic *vioapic;

	vioapic = vm_ioapic(vm);
	error = vioapic_mmio_rw(vioapic, vcpuid, gpa, rval, size, true);

	return (error);
}

int
vioapic_mmio_write(void *vm, int vcpuid, uint64_t gpa, uint64_t wval,
    int size, UNUSED void *arg)
{
	int error;
	struct vioapic *vioapic;

	vioapic = vm_ioapic(vm);
	error = vioapic_mmio_rw(vioapic, vcpuid, gpa, &wval, size, false);
	return (error);
}

void
vioapic_process_eoi(struct vm *vm, UNUSED int vcpuid, int vector)
{
	struct vioapic *vioapic;
	int pin;

	KASSERT(vector >= 0 && vector < 256,
	    ("vioapic_process_eoi: invalid vector %d", vector));

	vioapic = vm_ioapic(vm);
	VIOAPIC_CTR1(vioapic, "ioapic processing eoi for vector %d", vector);

	/*
	 * XXX keep track of the pins associated with this vector instead
	 * of iterating on every single pin each time.
	 */
	VIOAPIC_LOCK(vioapic);
	for (pin = 0; pin < REDIR_ENTRIES; pin++) {
		if ((vioapic->rtbl[pin].reg & IOART_REM_IRR) == 0)
			continue;
		if ((vioapic->rtbl[pin].reg & IOART_INTVEC) != vector)
			continue;
		vioapic->rtbl[pin].reg &= ~((uint64_t) IOART_REM_IRR);
		if (vioapic->rtbl[pin].acnt > 0) {
			VIOAPIC_CTR2(vioapic, "ioapic pin%d: asserted at eoi, "
			    "acnt %d", pin, vioapic->rtbl[pin].acnt);
			vioapic_send_intr(vioapic, pin);
		}
	}
	VIOAPIC_UNLOCK(vioapic);
}

struct vioapic *
vioapic_init(struct vm *vm)
{
	int i;
	struct vioapic *vioapic;

	vioapic = malloc(sizeof(struct vioapic));
	assert(vioapic);
	bzero(vioapic, sizeof(struct vioapic));
	vioapic->vm = vm;

	VIOAPIC_LOCK_INIT(vioapic);

	/* Initialize all redirection entries to mask all interrupts */
	for (i = 0; i < REDIR_ENTRIES; i++)
		vioapic->rtbl[i].reg = 0x0001000000010000UL;

	return (vioapic);
}

void
vioapic_cleanup(struct vioapic *vioapic)
{
	free(vioapic);
}

int
vioapic_pincount(UNUSED struct vm *vm)
{
	return (REDIR_ENTRIES);
}
