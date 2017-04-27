/*-
 * Copyright (c) 2014 Hudson River Trading LLC
 * Written by: John H. Baldwin <jhb@FreeBSD.org>
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
 * THIS SOFTWARE IS PROVIDED BY THE AUTHOR AND CONTRIBUTORS ``AS IS'' AND
 * ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED.  IN NO EVENT SHALL THE AUTHOR OR CONTRIBUTORS BE LIABLE
 * FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
 * DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
 * OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
 * HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
 * LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY
 * OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
 * SUCH DAMAGE.
 */

#include <xhyve/vmm/vmm_api.h>
#include <xhyve/ioapic.h>

/*
 * Assign PCI INTx interrupts to I/O APIC pins in a round-robin
 * fashion.  Note that we have no idea what the HPET is using, but the
 * HPET is also programmable whereas this is intended for hardwired
 * PCI interrupts.
 *
 * This assumes a single I/O APIC where pins >= 16 are permitted for
 * PCI devices.
 */
static int pci_pins;

void
ioapic_init(void)
{

	if (xh_vm_ioapic_pincount(&pci_pins) < 0) {
		pci_pins = 0;
		return;
	}

	/* Ignore the first 16 pins. */
	if (pci_pins <= 16) {
		pci_pins = 0;
		return;
	}
	pci_pins -= 16;
}

int
ioapic_pci_alloc_irq(void)
{
	static int last_pin;

	if (pci_pins == 0)
		return (-1);
	return (16 + (last_pin++ % pci_pins));
}
