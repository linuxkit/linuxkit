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
#include <errno.h>
#include <sys/sysctl.h>
#include <Hypervisor/hv.h>
#include <Hypervisor/hv_vmx.h>
#include <xhyve/support/misc.h>
#include <xhyve/support/specialreg.h>
#include <xhyve/vmm/vmm.h>
#include <xhyve/vmm/intel/vmx.h>
#include <xhyve/vmm/intel/vmx_msr.h>

static bool
vmx_ctl_allows_one_setting(uint64_t msr_val, int bitpos)
{
	if (msr_val & (1UL << (bitpos + 32)))
		return (TRUE);
	else
		return (FALSE);
}

static bool
vmx_ctl_allows_zero_setting(uint64_t msr_val, int bitpos)
{
	if ((msr_val & (1UL << bitpos)) == 0)
		return (TRUE);
	else
		return (FALSE);
}

int vmx_set_ctlreg(hv_vmx_capability_t cap_field, uint32_t ones_mask,
	uint32_t zeros_mask, uint32_t *retval)
{
	int i;
	uint64_t cap;
	bool one_allowed, zero_allowed;

	/* We cannot ask the same bit to be set to both '1' and '0' */
	if ((ones_mask ^ zeros_mask) != (ones_mask | zeros_mask)) {
		return EINVAL;
	}

	if (hv_vmx_read_capability(cap_field, &cap)) {
		return EINVAL;
	}

	for (i = 0; i < 32; i++) {
		one_allowed = vmx_ctl_allows_one_setting(cap, i);
		zero_allowed = vmx_ctl_allows_zero_setting(cap, i);

		if (zero_allowed && !one_allowed) {
			/* must be zero */
			if (ones_mask & (1 << i)) {
				fprintf(stderr,
					"vmx_set_ctlreg: cap_field: %d bit: %d must be zero\n",
					cap_field, i);
				return (EINVAL);
			}
			*retval &= ~(1 << i);
		} else if (one_allowed && !zero_allowed) {
			/* must be one */
			if (zeros_mask & (1 << i)) {
				fprintf(stderr,
					"vmx_set_ctlreg: cap_field: %d bit: %d must be one\n",
					cap_field, i);
				return (EINVAL);
			}
			*retval |= 1 << i;
		} else {
			/* don't care */
			if (zeros_mask & (1 << i)){
				*retval &= ~(1 << i);
			} else if (ones_mask & (1 << i)) {
				*retval |= 1 << i;
			} else {
				/* XXX: don't allow unspecified don't cares */
				fprintf(stderr,
					"vmx_set_ctlreg: cap_field: %d bit: %d unspecified "
					"don't care\n", cap_field, i);
				return (EINVAL);
			}
		}
	}

	return (0);
}

static uint64_t misc_enable;
static uint64_t platform_info;
static uint64_t turbo_ratio_limit;

static bool
pat_valid(uint64_t val)
{
	int i, pa;

	/*
	 * From Intel SDM: Table "Memory Types That Can Be Encoded With PAT"
	 *
	 * Extract PA0 through PA7 and validate that each one encodes a
	 * valid memory type.
	 */
	for (i = 0; i < 8; i++) {
		pa = (val >> (i * 8)) & 0xff;
		if (pa == 2 || pa == 3 || pa >= 8)
			return (false);
	}
	return (true);
}

void
vmx_msr_init(void) {
	uint64_t bus_freq, tsc_freq, ratio;
	size_t length;
	int i;

	length = sizeof(uint64_t);

	if (sysctlbyname("machdep.tsc.frequency", &tsc_freq, &length, NULL, 0)) {
	  xhyve_abort("machdep.tsc.frequency\n");
	}

	if (sysctlbyname("hw.busfrequency", &bus_freq, &length, NULL, 0)) {
	  xhyve_abort("hw.busfrequency\n");
	}

	/* Initialize emulated MSRs */
	/* FIXME */
	misc_enable = 1;
	/*
	 * Set mandatory bits
	 *  11:   branch trace disabled
	 *  12:   PEBS unavailable
	 * Clear unsupported features
	 *  16:   SpeedStep enable
	 *  18:   enable MONITOR FSM
	 */
	misc_enable |= (1u << 12) | (1u << 11);
	misc_enable &= ~((1u << 18) | (1u << 16));

	/*
	 * XXXtime
	 * The ratio should really be based on the virtual TSC frequency as
	 * opposed to the host TSC.
	 */
	ratio = (tsc_freq / bus_freq) & 0xff;

	/*
	 * The register definition is based on the micro-architecture
	 * but the following bits are always the same:
	 * [15:8]  Maximum Non-Turbo Ratio
	 * [28]    Programmable Ratio Limit for Turbo Mode
	 * [29]    Programmable TDC-TDP Limit for Turbo Mode
	 * [47:40] Maximum Efficiency Ratio
	 *
	 * The other bits can be safely set to 0 on all
	 * micro-architectures up to Haswell.
	 */
	platform_info = (ratio << 8) | (ratio << 40);

	/*
	 * The number of valid bits in the MSR_TURBO_RATIO_LIMITx register is
	 * dependent on the maximum cores per package supported by the micro-
	 * architecture. For e.g., Westmere supports 6 cores per package and
	 * uses the low 48 bits. Sandybridge support 8 cores per package and
	 * uses up all 64 bits.
	 *
	 * However, the unused bits are reserved so we pretend that all bits
	 * in this MSR are valid.
	 */
	for (i = 0; i < 8; i++) {
	  turbo_ratio_limit = (turbo_ratio_limit << 8) | ratio;
	}
}

void
vmx_msr_guest_init(struct vmx *vmx, int vcpuid)
{
	uint64_t *guest_msrs;

	guest_msrs = vmx->guest_msrs[vcpuid];


	hv_vcpu_enable_native_msr(((hv_vcpuid_t) vcpuid), MSR_LSTAR, 1);
	hv_vcpu_enable_native_msr(((hv_vcpuid_t) vcpuid), MSR_CSTAR, 1);
	hv_vcpu_enable_native_msr(((hv_vcpuid_t) vcpuid), MSR_STAR, 1);
	hv_vcpu_enable_native_msr(((hv_vcpuid_t) vcpuid), MSR_SF_MASK, 1);
	hv_vcpu_enable_native_msr(((hv_vcpuid_t) vcpuid), MSR_KGSBASE, 1);

	/*
	 * Initialize guest IA32_PAT MSR with default value after reset.
	 */
	guest_msrs[IDX_MSR_PAT] = PAT_VALUE(0, PAT_WRITE_BACK) |
		PAT_VALUE(1, PAT_WRITE_THROUGH) |
		PAT_VALUE(2, PAT_UNCACHED)      |
		PAT_VALUE(3, PAT_UNCACHEABLE)   |
		PAT_VALUE(4, PAT_WRITE_BACK)    |
		PAT_VALUE(5, PAT_WRITE_THROUGH) |
		PAT_VALUE(6, PAT_UNCACHED)      |
		PAT_VALUE(7, PAT_UNCACHEABLE);

	return;
}

int
vmx_rdmsr(struct vmx *vmx, int vcpuid, u_int num, uint64_t *val)
{
	const uint64_t *guest_msrs;
	int error;

	guest_msrs = vmx->guest_msrs[vcpuid];
	error = 0;

	switch (num) {
	case MSR_EFER:
		*val = vmcs_read(vcpuid, VMCS_GUEST_IA32_EFER);
		break;
	case MSR_MCG_CAP:
	case MSR_MCG_STATUS:
		*val = 0;
		break;
	case MSR_MTRRcap:
	case MSR_MTRRdefType:
	case MSR_MTRR4kBase:
	case MSR_MTRR4kBase + 1:
	case MSR_MTRR4kBase + 2:
	case MSR_MTRR4kBase + 3:
	case MSR_MTRR4kBase + 4:
	case MSR_MTRR4kBase + 5:
	case MSR_MTRR4kBase + 6:
	case MSR_MTRR4kBase + 7:
	case MSR_MTRR4kBase + 8:
	case MSR_MTRR16kBase:
	case MSR_MTRR16kBase + 1:
	case MSR_MTRR64kBase:
		*val = 0;
		break;
	case MSR_IA32_MISC_ENABLE:
		*val = misc_enable;
		break;
	case MSR_PLATFORM_INFO:
		*val = platform_info;
		break;
	case MSR_TURBO_RATIO_LIMIT:
	case MSR_TURBO_RATIO_LIMIT1:
		*val = turbo_ratio_limit;
		break;
	case MSR_PAT:
		*val = guest_msrs[IDX_MSR_PAT];
		break;
	default:
		error = EINVAL;
		break;
	}
	return (error);
}

int
vmx_wrmsr(struct vmx *vmx, int vcpuid, u_int num, uint64_t val)
{
	uint64_t *guest_msrs;
	uint64_t changed;
	int error;

	guest_msrs = vmx->guest_msrs[vcpuid];
	error = 0;

	switch (num) {
	case MSR_EFER:
		vmcs_write(vcpuid, VMCS_GUEST_IA32_EFER, val);
		break;
	case MSR_MCG_CAP:
	case MSR_MCG_STATUS:
		break;      /* ignore writes */
	case MSR_MTRRcap:
		vm_inject_gp(vmx->vm, vcpuid);
		break;
	case MSR_MTRRdefType:
	case MSR_MTRR4kBase:
	case MSR_MTRR4kBase + 1:
	case MSR_MTRR4kBase + 2:
	case MSR_MTRR4kBase + 3:
	case MSR_MTRR4kBase + 4:
	case MSR_MTRR4kBase + 5:
	case MSR_MTRR4kBase + 6:
	case MSR_MTRR4kBase + 7:
	case MSR_MTRR4kBase + 8:
	case MSR_MTRR16kBase:
	case MSR_MTRR16kBase + 1:
	case MSR_MTRR64kBase:
		break;      /* Ignore writes */
	case MSR_IA32_MISC_ENABLE:
		changed = val ^ misc_enable;
		/*
		 * If the host has disabled the NX feature then the guest
		 * also cannot use it. However, a Linux guest will try to
		 * enable the NX feature by writing to the MISC_ENABLE MSR.
		 *
		 * This can be safely ignored because the memory management
		 * code looks at CPUID.80000001H:EDX.NX to check if the
		 * functionality is actually enabled.
		 */
		changed &= ~(1UL << 34);

		/*
		 * Punt to userspace if any other bits are being modified.
		 */
		if (changed)
			error = EINVAL;

		break;
	case MSR_PAT:
		if (pat_valid(val))
			guest_msrs[IDX_MSR_PAT] = val;
		else
			vm_inject_gp(vmx->vm, vcpuid);
		break;
	default:
		error = EINVAL;
		break;
	}

	return (error);
}
