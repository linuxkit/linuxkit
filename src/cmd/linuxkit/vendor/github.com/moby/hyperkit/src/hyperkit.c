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
#include <stdlib.h>
#include <string.h>
#include <err.h>
#include <fcntl.h>
#include <libgen.h>
#include <unistd.h>
#include <assert.h>
#include <errno.h>
#include <pthread.h>
#include <sysexits.h>
#include <ctype.h>
#include <inttypes.h>
#include <signal.h>
#include <sys/types.h>
#include <sys/mman.h>
#include <sys/time.h>
#include <sys/param.h>

#include <dispatch/dispatch.h>

#include <xhyve/support/misc.h>
#include <xhyve/support/atomic.h>
#include <xhyve/support/segments.h>
#include <xhyve/support/cpuset.h>
#include <xhyve/vmm/vmm_api.h>

#include <xhyve/xhyve.h>
#include <xhyve/acpi.h>
#include <xhyve/inout.h>
#include <xhyve/dbgport.h>
#include <xhyve/ioapic.h>
#include <xhyve/mem.h>
#include <xhyve/mevent.h>
#include <xhyve/mptbl.h>
#include <xhyve/pci_emul.h>
#include <xhyve/pci_irq.h>
#include <xhyve/pci_lpc.h>
#include <xhyve/smbiostbl.h>
#include <xhyve/xmsr.h>
#include <xhyve/rtc.h>
#include <xhyve/fwctl.h>

#include <xhyve/firmware/kexec.h>
#include <xhyve/firmware/fbsd.h>
#include <xhyve/firmware/bootrom.h>
#include <xhyve/firmware/multiboot.h>

#ifdef HAVE_OCAML
#include <caml/callback.h>
#include <caml/threads.h>
#endif

#define GUEST_NIO_PORT 0x488 /* guest upcalls via i/o port */

#define MB (1024UL * 1024)

typedef int (*vmexit_handler_t)(struct vm_exit *, int *vcpu);
extern int vmexit_task_switch(struct vm_exit *, int *vcpu);

char *vmname = "vm";

int guest_ncpus;
char *guest_uuid_str;
static char *pidfile;

static int guest_vmexit_on_hlt, guest_vmexit_on_pause;
static int virtio_msix = 1;
static int x2apic_mode = 0;	/* default is xAPIC */

static int strictio;
static int strictmsr = 1;

static int acpi;

static char *progname;
static const int BSP = 0;

static cpuset_t cpumask;

static void vcpu_loop(int vcpu, uint64_t rip);

static struct vm_exit vmexit[VM_MAXCPU];

static struct bhyvestats {
	uint64_t vmexit_bogus;
	uint64_t vmexit_bogus_switch;
	uint64_t vmexit_hlt;
	uint64_t vmexit_pause;
	uint64_t vmexit_mtrap;
	uint64_t vmexit_inst_emul;
	uint64_t cpu_switch_rotate;
	uint64_t cpu_switch_direct;
} stats;

static struct mt_vmm_info {
	pthread_t mt_thr;
	int mt_vcpu;
} mt_vmm_info[VM_MAXCPU];

static uint64_t (*fw_func)(void);

__attribute__ ((noreturn)) static void
usage(int code)
{

        fprintf(stderr,
                "Usage: %s [-behuwxMACHPWY] [-c vcpus] [-F <pidfile>] [-g <gdb port>] [-l <lpc>]\n"
		"       %*s [-m mem] [-p vcpu:hostcpu] [-s <pci>] [-U uuid] -f <fw>\n"
		"       -A: create ACPI tables\n"
		"       -c: # cpus (default 1)\n"
		"       -C: include guest memory in core file\n"
		"       -e: exit on unhandled I/O access\n"
		"       -f: firmware\n"
		"       -F: pidfile\n"
		"       -g: gdb port\n"
		"       -h: help\n"
		"       -H: vmexit from the guest on hlt\n"
		"       -l: LPC device configuration. Ex: -l com1,stdio -l com2,autopty -l com2,/dev/myownpty\n"
		"       -m: memory size in MB, may be suffixed with one of K, M, G or T\n"
		"       -P: vmexit from the guest on pause\n"
		"       -s: <slot,driver,configinfo> PCI slot config\n"
		"       -u: RTC keeps UTC time\n"
		"       -U: uuid\n"
		"       -v: show build version\n"
		"       -w: ignore unimplemented MSRs\n"
		"       -W: force virtio to use single-vector MSI\n"
		"       -x: local apic is in x2APIC mode\n"
		"       -Y: disable MPtable generation\n",
		progname, (int)strlen(progname), "");

	exit(code);
}

__attribute__ ((noreturn)) static void
show_version()
{
        fprintf(stderr, "%s: %s\n\n%s\n",progname, VERSION,
		"Homepage: https://github.com/docker/hyperkit\n"
		"License: BSD\n");
		exit(0);
}

void
xh_vm_inject_fault(int vcpu, int vector, int errcode_valid,
    uint32_t errcode)
{
	int error, restart_instruction;

	restart_instruction = 1;

	error = xh_vm_inject_exception(vcpu, vector, errcode_valid, errcode,
	    restart_instruction);
	assert(error == 0);
}

void *
paddr_guest2host(uintptr_t gaddr, size_t len)
{
	return (xh_vm_map_gpa(gaddr, len));
}

int
fbsdrun_vmexit_on_pause(void)
{
	return (guest_vmexit_on_pause);
}

int
fbsdrun_vmexit_on_hlt(void)
{
	return (guest_vmexit_on_hlt);
}

int
fbsdrun_virtio_msix(void)
{
	return (virtio_msix);
}

static void
spinup_ap_realmode(int newcpu, uint64_t *rip)
{
	int vector, error;
	uint16_t cs;
	uint64_t desc_base;
	uint32_t desc_limit, desc_access;

	vector = (int) (*rip >> XHYVE_PAGE_SHIFT);
	*rip = 0;

	/*
	 * Update the %cs and %rip of the guest so that it starts
	 * executing real mode code at at 'vector << 12'.
	 */
	error = xh_vm_set_register(newcpu, VM_REG_GUEST_RIP, *rip);
	assert(error == 0);

	error = xh_vm_get_desc(newcpu, VM_REG_GUEST_CS, &desc_base, &desc_limit,
		&desc_access);
	assert(error == 0);

	desc_base = (uint64_t) (vector << XHYVE_PAGE_SHIFT);
	error = xh_vm_set_desc(newcpu, VM_REG_GUEST_CS, desc_base, desc_limit,
		desc_access);
	assert(error == 0);

	cs = (uint16_t) ((vector << XHYVE_PAGE_SHIFT) >> 4);
	error = xh_vm_set_register(newcpu, VM_REG_GUEST_CS, cs);
	assert(error == 0);
}

static void *
vcpu_thread(void *param)
{
	struct mt_vmm_info *mtp;
	uint64_t rip_entry;
	int vcpu;
	int error;
	char ident[16];

	mtp = param;
	vcpu = mtp->mt_vcpu;
	rip_entry = 0xfff0;

	snprintf(ident, sizeof(ident), "vcpu:%d", vcpu);
	pthread_setname_np(ident);

	error = xh_vcpu_create(vcpu);
	assert(error == 0);

	vcpu_set_capabilities(vcpu);

	error = xh_vcpu_reset(vcpu);
	assert(error == 0);

	if (vcpu == BSP) {
		rip_entry = fw_func();
	} else {
		rip_entry = vmexit[vcpu].rip;
		spinup_ap_realmode(vcpu, &rip_entry);
	}

	vmexit[vcpu].rip = rip_entry;
	vmexit[vcpu].inst_length = 0;

	vcpu_loop(vcpu, vmexit[vcpu].rip);

	/* not reached */
	exit(1);
	return (NULL);
}

void
vcpu_add(int fromcpu, int newcpu, uint64_t rip)
{
	int error;

	assert(fromcpu == BSP);

	/*
	 * The 'newcpu' must be activated in the context of 'fromcpu'. If
	 * vm_activate_cpu() is delayed until newcpu's pthread starts running
	 * then vmm.ko is out-of-sync with bhyve and this can create a race
	 * with vm_suspend().
	 */
	error = xh_vm_activate_cpu(newcpu);
	assert(error == 0);

	CPU_SET_ATOMIC(((unsigned) newcpu), &cpumask);

	mt_vmm_info[newcpu].mt_vcpu = newcpu;

	vmexit[newcpu].rip = rip;

	error = pthread_create(&mt_vmm_info[newcpu].mt_thr, NULL, vcpu_thread,
		&mt_vmm_info[newcpu]);

	assert(error == 0);
}

static int
vcpu_delete(int vcpu)
{
	if (!CPU_ISSET(((unsigned) vcpu), &cpumask)) {
		fprintf(stderr, "Attempting to delete unknown cpu %d\n", vcpu);
		exit(1);
	}

	CPU_CLR_ATOMIC(((unsigned) vcpu), &cpumask);
	return (CPU_EMPTY(&cpumask));
}

static int
vmexit_handle_notify(UNUSED struct vm_exit *vme, UNUSED int *pvcpu,
	UNUSED uint32_t eax)
{
	return (VMEXIT_CONTINUE);
}

static int
vmexit_inout(struct vm_exit *vme, int *pvcpu)
{
	int error;
	int bytes, port, in, out, string;
	int vcpu;

	vcpu = *pvcpu;

	port = vme->u.inout.port;
	bytes = vme->u.inout.bytes;
	string = vme->u.inout.string;
	in = vme->u.inout.in;
	out = !in;

	/* Extra-special case of host notifications */
	if (out && port == GUEST_NIO_PORT) {
		error = vmexit_handle_notify(vme, pvcpu, vme->u.inout.eax);
		return (error);
	}

	error = emulate_inout(vcpu, vme, strictio);
	if (error) {
		fprintf(stderr, "Unhandled %s%c 0x%04x at 0x%llx\n",
			in ? "in" : "out",
			bytes == 1 ? 'b' : (bytes == 2 ? 'w' : 'l'),
			port, vmexit->rip);
		return (VMEXIT_ABORT);
	} else {
		return (VMEXIT_CONTINUE);
	}
}

static int
vmexit_rdmsr(struct vm_exit *vme, int *pvcpu)
{
	uint64_t val;
	uint32_t eax, edx;
	int error;

	val = 0;
	error = emulate_rdmsr(*pvcpu, vme->u.msr.code, &val);
	if (error != 0) {
		fprintf(stderr, "rdmsr to register %#x on vcpu %d\n",
		    vme->u.msr.code, *pvcpu);
		if (strictmsr) {
			vm_inject_gp(*pvcpu);
			return (VMEXIT_CONTINUE);
		}
	}

	eax = (uint32_t) val;
	error = xh_vm_set_register(*pvcpu, VM_REG_GUEST_RAX, eax);
	assert(error == 0);

	edx = val >> 32;
	error = xh_vm_set_register(*pvcpu, VM_REG_GUEST_RDX, edx);
	assert(error == 0);

	return (VMEXIT_CONTINUE);
}

static int
vmexit_wrmsr(struct vm_exit *vme, int *pvcpu)
{
	int error;

	error = emulate_wrmsr(*pvcpu, vme->u.msr.code, vme->u.msr.wval);
	if (error != 0) {
		fprintf(stderr, "wrmsr to register %#x(%#llx) on vcpu %d\n",
		    vme->u.msr.code, vme->u.msr.wval, *pvcpu);
		if (strictmsr) {
			vm_inject_gp(*pvcpu);
			return (VMEXIT_CONTINUE);
		}
	}
	return (VMEXIT_CONTINUE);
}

static int
vmexit_spinup_ap(struct vm_exit *vme, int *pvcpu)
{
	assert(vme->u.spinup_ap.vcpu != 0);
	assert(vme->u.spinup_ap.vcpu < guest_ncpus);

	vcpu_add(*pvcpu, vme->u.spinup_ap.vcpu, vme->u.spinup_ap.rip);

	return (VMEXIT_CONTINUE);
}

static int
vmexit_vmx(struct vm_exit *vme, int *pvcpu)
{
	fprintf(stderr, "vm exit[%d]\n", *pvcpu);
	fprintf(stderr, "\treason\t\tVMX\n");
	fprintf(stderr, "\trip\t\t0x%016llx\n", vme->rip);
	fprintf(stderr, "\tinst_length\t%d\n", vme->inst_length);
	fprintf(stderr, "\tstatus\t\t%d\n", vme->u.vmx.status);
	fprintf(stderr, "\texit_reason\t%u\n", vme->u.vmx.exit_reason);
	fprintf(stderr, "\tqualification\t0x%016llx\n",
	    vme->u.vmx.exit_qualification);
	fprintf(stderr, "\tinst_type\t\t%d\n", vme->u.vmx.inst_type);
	fprintf(stderr, "\tinst_error\t\t%d\n", vme->u.vmx.inst_error);
	return (VMEXIT_ABORT);
}

static int
vmexit_bogus(struct vm_exit *vme, UNUSED int *pvcpu)
{
	assert(vme->inst_length == 0);

	stats.vmexit_bogus++;

	return (VMEXIT_CONTINUE);
}

static int
vmexit_hlt(UNUSED struct vm_exit *vme, UNUSED int *pvcpu)
{
	stats.vmexit_hlt++;

	/*
	 * Just continue execution with the next instruction. We use
	 * the HLT VM exit as a way to be friendly with the host
	 * scheduler.
	 */
	return (VMEXIT_CONTINUE);
}

static int
vmexit_pause(UNUSED struct vm_exit *vme, UNUSED int *pvcpu)
{
	stats.vmexit_pause++;

	return (VMEXIT_CONTINUE);
}

static int
vmexit_mtrap(struct vm_exit *vme, UNUSED int *pvcpu)
{
	assert(vme->inst_length == 0);

	stats.vmexit_mtrap++;

	return (VMEXIT_CONTINUE);
}

static int
vmexit_inst_emul(struct vm_exit *vme, int *pvcpu)
{
	int err, i;
	struct vie *vie;

	stats.vmexit_inst_emul++;

	vie = &vme->u.inst_emul.vie;
	err = emulate_mem(*pvcpu, vme->u.inst_emul.gpa, vie,
		&vme->u.inst_emul.paging);

	if (err) {
		if (err == ESRCH) {
			fprintf(stderr, "Unhandled memory access to 0x%llx\n",
			    vme->u.inst_emul.gpa);
		}

		fprintf(stderr, "Failed to emulate instruction [");
		for (i = 0; i < vie->num_valid; i++) {
			fprintf(stderr, "0x%02x%s", vie->inst[i],
			    i != (vie->num_valid - 1) ? " " : "");
		}
		fprintf(stderr, "] at 0x%llx\n", vme->rip);
		return (VMEXIT_ABORT);
	}

	return (VMEXIT_CONTINUE);
}

static pthread_mutex_t resetcpu_mtx = PTHREAD_MUTEX_INITIALIZER;
static pthread_cond_t resetcpu_cond = PTHREAD_COND_INITIALIZER;

static int
vmexit_suspend(struct vm_exit *vme, int *pvcpu)
{
	enum vm_suspend_how how;

	how = vme->u.suspended.how;

	vcpu_delete(*pvcpu);

	if (*pvcpu != BSP) {
		pthread_mutex_lock(&resetcpu_mtx);
		pthread_cond_signal(&resetcpu_cond);
		pthread_mutex_unlock(&resetcpu_mtx);
		pthread_exit(NULL);
	}

	pthread_mutex_lock(&resetcpu_mtx);
	while (!CPU_EMPTY(&cpumask)) {
		pthread_cond_wait(&resetcpu_cond, &resetcpu_mtx);
	}
	pthread_mutex_unlock(&resetcpu_mtx);

	switch ((int) (how)) {
	case VM_SUSPEND_POWEROFF:
	case VM_SUSPEND_HALT:
		exit(0);
	case VM_SUSPEND_RESET:
		exit(2);
	case VM_SUSPEND_TRIPLEFAULT:
		exit(3);
	default:
		fprintf(stderr, "vmexit_suspend: invalid reason %d\n", how);
		exit(100);
	}
}

static vmexit_handler_t handler[VM_EXITCODE_MAX] = {
	[VM_EXITCODE_INOUT] = vmexit_inout,
	[VM_EXITCODE_INOUT_STR] = vmexit_inout,
	[VM_EXITCODE_VMX] = vmexit_vmx,
	[VM_EXITCODE_BOGUS] = vmexit_bogus,
	[VM_EXITCODE_RDMSR] = vmexit_rdmsr,
	[VM_EXITCODE_WRMSR] = vmexit_wrmsr,
	[VM_EXITCODE_MTRAP] = vmexit_mtrap,
	[VM_EXITCODE_INST_EMUL] = vmexit_inst_emul,
	[VM_EXITCODE_SPINUP_AP] = vmexit_spinup_ap,
	[VM_EXITCODE_SUSPENDED] = vmexit_suspend,
	[VM_EXITCODE_TASK_SWITCH] = vmexit_task_switch,
};

void
vcpu_set_capabilities(int cpu)
{
	int err, tmp;

	if (fbsdrun_vmexit_on_hlt()) {
		err = xh_vm_get_capability(cpu, VM_CAP_HALT_EXIT, &tmp);
		if (err < 0) {
			fprintf(stderr, "VM exit on HLT not supported\n");
			exit(1);
		}
		xh_vm_set_capability(cpu, VM_CAP_HALT_EXIT, 1);
		if (cpu == BSP)
			handler[VM_EXITCODE_HLT] = vmexit_hlt;
	}

        if (fbsdrun_vmexit_on_pause()) {
		/*
		 * pause exit support required for this mode
		 */
		err = xh_vm_get_capability(cpu, VM_CAP_PAUSE_EXIT, &tmp);
		if (err < 0) {
			fprintf(stderr,
			    "SMP mux requested, no pause support\n");
			exit(1);
		}
		xh_vm_set_capability(cpu, VM_CAP_PAUSE_EXIT, 1);
		if (cpu == BSP)
			handler[VM_EXITCODE_PAUSE] = vmexit_pause;
        }

	if (x2apic_mode)
		err = xh_vm_set_x2apic_state(cpu, X2APIC_ENABLED);
	else
		err = xh_vm_set_x2apic_state(cpu, X2APIC_DISABLED);

	if (err) {
		fprintf(stderr, "Unable to set x2apic state (%d)\n", err);
		exit(1);
	}
}

static void
vcpu_loop(int vcpu, uint64_t startrip)
{
	int error, rc, prevcpu;
	enum vm_exitcode exitcode;
	cpuset_t active_cpus;

	error = xh_vm_active_cpus(&active_cpus);
	assert(CPU_ISSET(((unsigned) vcpu), &active_cpus));

	error = xh_vm_set_register(vcpu, VM_REG_GUEST_RIP, startrip);
	assert(error == 0);

	while (1) {
		error = xh_vm_run(vcpu, &vmexit[vcpu]);
		if (error != 0)
			break;

		prevcpu = vcpu;

		exitcode = vmexit[vcpu].exitcode;
		if (exitcode >= VM_EXITCODE_MAX || handler[exitcode] == NULL) {
			fprintf(stderr, "vcpu_loop: unexpected exitcode 0x%x\n",
			    exitcode);
			exit(1);
		}

                rc = (*handler[exitcode])(&vmexit[vcpu], &vcpu);

		switch (rc) {
		case VMEXIT_CONTINUE:
			break;
		case VMEXIT_ABORT:
			xh_vm_vcpu_dump(vcpu);
			abort();
		default:
			exit(1);
		}
	}
	fprintf(stderr, "vm_run error %d, errno %d\n", error, errno);
}

static int
num_vcpus_allowed(void)
{
	return (VM_MAXCPU);
}

static int
expand_number(const char *buf, uint64_t *num)
{
	char *endptr;
	uintmax_t umaxval;
	uint64_t number;
	unsigned shift;
	int serrno;

	serrno = errno;
	errno = 0;
	umaxval = strtoumax(buf, &endptr, 0);
	if (umaxval > UINT64_MAX)
		errno = ERANGE;
	if (errno != 0)
		return (-1);
	errno = serrno;
	number = umaxval;

	switch (tolower((unsigned char)*endptr)) {
	case 'e':
		shift = 60;
		break;
	case 'p':
		shift = 50;
		break;
	case 't':
		shift = 40;
		break;
	case 'g':
		shift = 30;
		break;
	case 'm':
		shift = 20;
		break;
	case 'k':
		shift = 10;
		break;
	case 'b':
	case '\0': /* No unit. */
		*num = number;
		return (0);
	default:
		/* Unrecognized unit. */
		errno = EINVAL;
		return (-1);
	}

	if ((number << shift) >> shift != number) {
		/* Overflow */
		errno = ERANGE;
		return (-1);
	}
	*num = number << shift;
	return (0);
}

static int
parse_memsize(const char *opt, size_t *ret_memsize)
{
	char *endptr;
	size_t optval;
	int error;

	optval = strtoul(opt, &endptr, 0);
	if (*opt != '\0' && *endptr == '\0') {
		/*
		 * For the sake of backward compatibility if the memory size
		 * specified on the command line is less than a megabyte then
		 * it is interpreted as being in units of MB.
		 */
		if (optval < MB)
			optval *= MB;
		*ret_memsize = optval;
		error = 0;
	} else
		error = expand_number(opt, ((uint64_t *) ret_memsize));

	return (error);
}

static int
firmware_parse(const char *opt) {
	char *fw, *opt1 = NULL, *opt2 = NULL, *opt3 = NULL, *cp;

	fw = strdup(opt);

	if (strncmp(fw, "kexec", strlen("kexec")) == 0) {
		fw_func = kexec;
	} else if (strncmp(fw, "fbsd", strlen("fbsd")) == 0) {
		fw_func = fbsd_load;
	} else if (strncmp(fw, "bootrom", strlen("bootrom")) == 0) {
		fw_func = bootrom_load;
	} else if (strncmp(fw, "multiboot", strlen("multiboot")) == 0) {
		fw_func = multiboot;
	} else {
		goto fail;
	}

// Gets first comma-separated option from cur and stores it in next.
#define NEXTARG(cur, next, scratch) do {			\
	if (cur && (scratch = strchr(cur, ',')) != NULL) {	\
		*scratch = '\0';				\
		next = scratch + 1;				\
	}							\
} while(0)

	NEXTARG(fw, opt1, cp);
	NEXTARG(opt1, opt2, cp);
	NEXTARG(opt2, opt3, cp);

#undef NEXTARG

	// Replace zero length options with NULLs
	opt1 = opt1 && strlen(opt1) ? opt1 : NULL;
	opt2 = opt2 && strlen(opt2) ? opt2 : NULL;
	opt3 = opt3 && strlen(opt3) ? opt3 : NULL;

	int ret = 1;
	if (fw_func == kexec) {
		ret = kexec_init(opt1, opt2, opt3);
	} else if (fw_func == fbsd_load) {
		/* FIXME: let user set boot-loader serial device */
		ret = fbsd_init(opt1, opt2, opt3, NULL);
	} else if (fw_func == bootrom_load) {
		ret = bootrom_init(opt1);
	} else if (fw_func == multiboot) {
		ret = multiboot_init(opt1, opt2, opt3);
	}
	if (ret)
		goto fail;

	return 0;

fail:
	fprintf(stderr, "Invalid firmware argument\n"
		"    -f kexec,'kernel'[,'initrd'][,'\"cmdline\"']\n"
		"    -f fbsd,'userboot','boot volume'[,'\"kernel env\"']\n"
		"    -f bootrom,'ROM'\n"
		"    -f multiboot,'kernel'[,module[;cmdline][:module[;cmdline]]...][,cmdline]\n");

	return -1;
}

static void
remove_pidfile()
{
	int error;

	if (pidfile == NULL)
		return;

	error = unlink(pidfile);
	if (error < 0)
		fprintf(stderr, "Failed to remove pidfile\n");
}

static int
setup_pidfile()
{
	int f, error, pid;
	char pid_str[21];

	if (pidfile == NULL)
		return 0;

	pid = getpid();

	error = sprintf(pid_str, "%d", pid);
	if (error < 0)
		goto fail;

	f = open(pidfile, O_CREAT|O_EXCL|O_WRONLY, 0644);
	if (f < 0)
		goto fail;

	error = atexit(remove_pidfile);
	if (error < 0) {
		close(f);
		remove_pidfile();
		goto fail;
	}

	if (0 > (write(f, (void*)pid_str, strlen(pid_str)))) {
		close(f);
		goto fail;
	}

	error = close(f);
	if (error < 0)
		goto fail;

	return 0;

fail:
	fprintf(stderr, "Failed to set up pidfile\n");
	return -1;
}

int
main(int argc, char *argv[])
{
	int c, error, gdb_port, bvmcons, fw;
	int dump_guest_memory, max_vcpus, mptgen;
	int rtc_localtime;
	uint64_t rip;
	size_t memsize;
	struct sigaction sa_ign;

	bvmcons = 0;
	dump_guest_memory = 0;
	progname = basename(argv[0]);
	gdb_port = 0;
	guest_ncpus = 1;
	memsize = 256 * MB;
	mptgen = 1;
	rtc_localtime = 1;
	fw = 0;

	while ((c = getopt(argc, argv, "behvuwxMACHPWY:f:F:g:c:s:m:l:U:")) != -1) {
		switch (c) {
		case 'A':
			acpi = 1;
			break;
		case 'b':
			bvmcons = 1;
			break;
		case 'c':
			guest_ncpus = atoi(optarg);
			break;
		case 'C':
			dump_guest_memory = 1;
			break;
		case 'f':
			if (firmware_parse(optarg) != 0) {
				exit (1);
			} else {
				fw = 1;
				break;
			}
		case 'F':
			pidfile = optarg;
			break;
		case 'g':
			gdb_port = atoi(optarg);
			break;
		case 'l':
			if (lpc_device_parse(optarg) != 0) {
				errx(EX_USAGE, "invalid lpc device "
				    "configuration '%s'", optarg);
			}
			break;
		case 's':
			if (pci_parse_slot(optarg) != 0)
				exit(1);
			else
				break;
		case 'm':
			error = parse_memsize(optarg, &memsize);
			if (error)
				errx(EX_USAGE, "invalid memsize '%s'", optarg);
			break;
		case 'H':
			guest_vmexit_on_hlt = 1;
			break;
		case 'P':
			guest_vmexit_on_pause = 1;
			break;
		case 'e':
			strictio = 1;
			break;
		case 'u':
			rtc_localtime = 0;
			break;
		case 'U':
			guest_uuid_str = optarg;
			break;
		case 'w':
			strictmsr = 0;
			break;
		case 'W':
			virtio_msix = 0;
			break;
		case 'x':
			x2apic_mode = 1;
			break;
		case 'Y':
			mptgen = 0;
			break;
		case 'v':
			show_version();
		case 'h':
			usage(0);
		default:
			usage(1);
		}
	}

	if (fw != 1)
		usage(1);

	/*
	 * We don't want SIGPIPEs ever, be sure to do this before any threads
	 * are created.
	 */
	sa_ign.sa_handler = SIG_IGN;
	sa_ign.sa_flags = 0;
	error = sigaction(SIGPIPE, &sa_ign, NULL);
	if (error) {
		perror("sigaction(SIGPIPE)");
		exit(1);
	}

#ifdef HAVE_OCAML
	caml_startup(argv) ;
	caml_release_runtime_system();
#endif
	error = xh_vm_create();
	if (error) {
		fprintf(stderr, "Unable to create VM (%d)\n", error);
		exit(1);
	}

	if (guest_ncpus < 1) {
		fprintf(stderr, "Invalid guest vCPUs (%d)\n", guest_ncpus);
		exit(1);
	}

	max_vcpus = num_vcpus_allowed();
	if (guest_ncpus > max_vcpus) {
		fprintf(stderr, "%d vCPUs requested but only %d available\n",
			guest_ncpus, max_vcpus);
		exit(1);
	}

	error = xh_vm_setup_memory(memsize, VM_MMAP_ALL);
	if (error) {
		fprintf(stderr, "Unable to setup memory (%d)\n", error);
		exit(1);
	}

	error = init_msr();
	if (error) {
		fprintf(stderr, "init_msr error %d\n", error);
		exit(1);
	}

	error = setup_pidfile();
	if (error) {
		fprintf(stderr, "pidfile error %d\n", error);
		exit(1);
	}

	init_mem();
	init_inout();
	pci_irq_init();
	ioapic_init();

	rtc_init(rtc_localtime);
	sci_init();

	/*
	 * Exit if a device emulation finds an error in it's initilization
	 */
	if (init_pci() != 0)
		exit(1);

	if (gdb_port != 0)
		init_dbgport(gdb_port);

	if (bvmcons)
		init_bvmcons();

	/*
	 * build the guest tables, MP etc.
	 */
	if (mptgen) {
		error = mptable_build(guest_ncpus);
		if (error)
			exit(1);
	}

	error = smbios_build();
	assert(error == 0);

	if (acpi) {
		error = acpi_build(guest_ncpus);
		assert(error == 0);
	}

	if (bootrom()) {
		fwctl_init();
	}

	rip = 0;

	// Use GCD to register signal handlers. These are not reentrant, so can call xhyve directly
	dispatch_source_t sigusr1_source = dispatch_source_create(DISPATCH_SOURCE_TYPE_SIGNAL, SIGUSR1, 0, dispatch_get_global_queue(0, 0));
	dispatch_source_t sigusr2_source = dispatch_source_create(DISPATCH_SOURCE_TYPE_SIGNAL, SIGUSR2, 0, dispatch_get_global_queue(0, 0));

	dispatch_source_set_event_handler(sigusr1_source, ^{
			fprintf(stdout, "received sigusr1, pausing\n");
			xh_hv_pause(1);
		});
	dispatch_source_set_event_handler(sigusr2_source, ^{
			fprintf(stdout, "received sigusr2, unpausing\n");
			xh_hv_pause(0);
		});

	signal(SIGUSR1, SIG_IGN);
	signal(SIGUSR2, SIG_IGN);

	dispatch_resume(sigusr1_source);
	dispatch_resume(sigusr2_source);

	vcpu_add(BSP, BSP, rip);

	/*
	 * Head off to the main event dispatch loop
	 */
	mevent_dispatch();

	exit(1);
}
