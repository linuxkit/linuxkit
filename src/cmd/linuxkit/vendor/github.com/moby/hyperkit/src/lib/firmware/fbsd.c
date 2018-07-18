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

/*-
 * Copyright (c) 2011 Google, Inc.
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
 *
 * $FreeBSD$
 */

#include <dirent.h>
#include <dlfcn.h>
#include <errno.h>
#include <err.h>
#include <fcntl.h>
#include <getopt.h>
#include <libgen.h>
#include <limits.h>
#include <stdio.h>
#include <stdlib.h>
#include <setjmp.h>
#include <string.h>
#include <sysexits.h>
#include <termios.h>
#include <unistd.h>
#include <assert.h>
#include <sys/ioctl.h>
#include <sys/stat.h>
#include <sys/disk.h>
#include <sys/queue.h>
#include <xhyve/support/misc.h>
#include <xhyve/support/segments.h>
#include <xhyve/support/specialreg.h>
#include <xhyve/vmm/vmm_api.h>
#include <xhyve/firmware/fbsd.h>

#define	I386_TSS_SIZE 104

#define	DESC_PRESENT 0x00000080
#define	DESC_DEF32 0x00004000
#define	DESC_GRAN 0x00008000
#define	DESC_UNUSABLE 0x00010000

#define	GUEST_CODE_SEL 1
#define	GUEST_DATA_SEL 2
#define	GUEST_TSS_SEL 3
#define	GUEST_GDTR_LIMIT64 (3 * 8 - 1)

#define	BSP 0
#define	NDISKS 32

static struct {
	char *userboot;
	char *bootvolume;
	char *kernelenv;
	char *cons;
} config;

static char *host_base;
static struct termios term, oldterm;
static int disk_fd[NDISKS];
static int ndisks;
static int consin_fd, consout_fd;
static jmp_buf exec_done;

static uint64_t vcpu_gdt_base, vcpu_cr3, vcpu_rsp, vcpu_rip;

typedef void (*func_t)(struct loader_callbacks *, void *, int, int);

static void cb_exit(void);

static struct segment_descriptor i386_gdt[] = {
	{ .sd_lolimit = 0, .sd_type = 0, /* NULL */
	  .sd_p = 0, .sd_hilimit = 0, .sd_def32 = 0, .sd_gran = 0},

	{ .sd_lolimit = 0xffff, .sd_type = SDT_MEMER, /* CODE */
	  .sd_p = 1, .sd_hilimit = 0xf, .sd_def32 = 1, .sd_gran = 1 },

	{ .sd_lolimit = 0xffff, .sd_type = SDT_MEMRW, /* DATA */
	  .sd_p = 1, .sd_hilimit = 0xf, .sd_def32 = 1, .sd_gran = 1 },

	{ .sd_lolimit = I386_TSS_SIZE - 1, /* TSS */
	  .sd_type = SDT_SYS386TSS, .sd_p = 1 }
};

static int
fbsd_set_regs_i386(uint32_t eip, uint32_t gdt_base, uint32_t esp)
{
	uint64_t cr0, rflags, desc_base;
	uint32_t desc_access, desc_limit, tss_base;
	uint16_t gsel;
	struct segment_descriptor *gdt;
	int error;

	cr0 = CR0_PE | CR0_NE;
	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_CR0, cr0)) != 0)
		goto done;

	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_CR4, 0)) != 0)
		goto done;

	/*
	 * Forcing EFER to 0 causes bhyve to clear the "IA-32e guest
	 * mode" entry control.
	 */
	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_EFER, 0)))
		goto done;

	gdt = xh_vm_map_gpa(gdt_base, 0x1000);
	if (gdt == NULL)
		return (EFAULT);
	memcpy(gdt, i386_gdt, sizeof(i386_gdt));
	desc_base = gdt_base;
	desc_limit = sizeof(i386_gdt) - 1;
	error = xh_vm_set_desc(BSP, VM_REG_GUEST_GDTR, desc_base, desc_limit, 0);
	if (error != 0)
		goto done;

	/* Place the TSS one page above the GDT. */
	tss_base = gdt_base + 0x1000;
	gdt[3].sd_lobase = tss_base;

	rflags = 0x2;
	error = xh_vm_set_register(BSP, VM_REG_GUEST_RFLAGS, rflags);
	if (error)
		goto done;

	desc_base = 0;
	desc_limit = 0xffffffff;
	desc_access = DESC_GRAN | DESC_DEF32 | DESC_PRESENT | SDT_MEMERA;
	error = xh_vm_set_desc(BSP, VM_REG_GUEST_CS, desc_base, desc_limit,
		desc_access);

	desc_access = DESC_GRAN | DESC_DEF32 | DESC_PRESENT | SDT_MEMRWA;
	error = xh_vm_set_desc(BSP, VM_REG_GUEST_DS, desc_base, desc_limit,
		desc_access);

	if (error)
		goto done;

	error = xh_vm_set_desc(BSP, VM_REG_GUEST_ES, desc_base, desc_limit,
		desc_access);

	if (error)
		goto done;

	error = xh_vm_set_desc(BSP, VM_REG_GUEST_FS, desc_base, desc_limit,
		desc_access);

	if (error)
		goto done;

	error = xh_vm_set_desc(BSP, VM_REG_GUEST_GS, desc_base, desc_limit,
		desc_access);

	if (error)
		goto done;

	error = xh_vm_set_desc(BSP, VM_REG_GUEST_SS, desc_base, desc_limit,
		desc_access);

	if (error)
		goto done;

	desc_base = tss_base;
	desc_limit = I386_TSS_SIZE - 1;
	desc_access = DESC_PRESENT | SDT_SYS386BSY;
	error = xh_vm_set_desc(BSP, VM_REG_GUEST_TR, desc_base, desc_limit,
		desc_access);

	if (error)
		goto done;


	error = xh_vm_set_desc(BSP, VM_REG_GUEST_LDTR, 0, 0, DESC_UNUSABLE);
	if (error)
		goto done;

	gsel = GSEL(GUEST_CODE_SEL, SEL_KPL);
	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_CS, gsel)) != 0)
		goto done;

	gsel = GSEL(GUEST_DATA_SEL, SEL_KPL);
	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_DS, gsel)) != 0)
		goto done;

	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_ES, gsel)) != 0)
		goto done;

	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_FS, gsel)) != 0)
		goto done;

	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_GS, gsel)) != 0)
		goto done;

	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_SS, gsel)) != 0)
		goto done;

	gsel = GSEL(GUEST_TSS_SEL, SEL_KPL);
	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_TR, gsel)) != 0)
		goto done;

	/* LDTR is pointing to the null selector */
	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_LDTR, 0)) != 0)
		goto done;

	/* entry point */
	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_RIP, eip)) != 0)
		goto done;

	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_RSP, esp)) != 0)
		goto done;

	error = 0;
done:
	return (error);
}

static int
fbsd_set_regs(uint64_t rip, uint64_t cr3, uint64_t gdt_base, uint64_t rsp)
{
	int error;
	uint64_t cr0, cr4, efer, rflags, desc_base;
	uint32_t desc_access, desc_limit;
	uint16_t gsel;

	cr0 = CR0_PE | CR0_PG | CR0_NE;
	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_CR0, cr0)) != 0)
		goto done;

	cr4 = CR4_PAE;
	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_CR4, cr4)) != 0)
		goto done;

	efer = EFER_LME | EFER_LMA;
	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_EFER, efer)))
		goto done;

	rflags = 0x2;
	error = xh_vm_set_register(BSP, VM_REG_GUEST_RFLAGS, rflags);
	if (error)
		goto done;

	desc_base = 0;
	desc_limit = 0;
	desc_access = 0x0000209B;
	error = xh_vm_set_desc(BSP, VM_REG_GUEST_CS, desc_base, desc_limit,
		desc_access);

	if (error)
		goto done;

	desc_access = 0x00000093;
	error = xh_vm_set_desc(BSP, VM_REG_GUEST_DS, desc_base, desc_limit,
		desc_access);

	if (error)
		goto done;

	error = xh_vm_set_desc(BSP, VM_REG_GUEST_ES, desc_base, desc_limit,
		desc_access);

	if (error)
		goto done;

	error = xh_vm_set_desc(BSP, VM_REG_GUEST_FS, desc_base, desc_limit,
		desc_access);

	if (error)
		goto done;

	error = xh_vm_set_desc(BSP, VM_REG_GUEST_GS, desc_base, desc_limit,
		desc_access);

	if (error)
		goto done;

	error = xh_vm_set_desc(BSP, VM_REG_GUEST_SS, desc_base, desc_limit,
		desc_access);

	if (error)
		goto done;

	/*
	 * XXX TR is pointing to null selector even though we set the
	 * TSS segment to be usable with a base address and limit of 0.
	 */
	desc_access = 0x0000008b;
	error = xh_vm_set_desc(BSP, VM_REG_GUEST_TR, 0, 0, desc_access);
	if (error)
		goto done;

	error = xh_vm_set_desc(BSP, VM_REG_GUEST_LDTR, 0, 0, DESC_UNUSABLE);
	if (error)
		goto done;

	gsel = GSEL(GUEST_CODE_SEL, SEL_KPL);
	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_CS, gsel)) != 0)
		goto done;

	gsel = GSEL(GUEST_DATA_SEL, SEL_KPL);
	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_DS, gsel)) != 0)
		goto done;

	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_ES, gsel)) != 0)
		goto done;

	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_FS, gsel)) != 0)
		goto done;

	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_GS, gsel)) != 0)
		goto done;

	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_SS, gsel)) != 0)
		goto done;

	/* XXX TR is pointing to the null selector */
	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_TR, 0)) != 0)
		goto done;

	/* LDTR is pointing to the null selector */
	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_LDTR, 0)) != 0)
		goto done;

	/* entry point */
	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_RIP, rip)) != 0)
		goto done;

	/* page table base */
	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_CR3, cr3)) != 0)
		goto done;

	desc_base = gdt_base;
	desc_limit = GUEST_GDTR_LIMIT64;
	error = xh_vm_set_desc(BSP, VM_REG_GUEST_GDTR, desc_base, desc_limit, 0);
	if (error != 0)
		goto done;

	if ((error = xh_vm_set_register(BSP, VM_REG_GUEST_RSP, rsp)) != 0)
		goto done;

	error = 0;
done:
	return (error);
}

/*
 * Console i/o callbacks
 */

static void
cb_putc(UNUSED void *arg, int ch)
{
	char c = (char) ch;

	(void) write(consout_fd, &c, 1);
}

static int
cb_getc(UNUSED void *arg)
{
	char c;

	if (read(consin_fd, &c, 1) == 1)
		return (c);
	return (-1);
}

static int
cb_poll(UNUSED void *arg)
{
	int n;

	if (ioctl(consin_fd, FIONREAD, &n) >= 0)
		return (n > 0);
	return (0);
}

/*
 * Host filesystem i/o callbacks
 */

struct cb_file {
	int cf_isdir;
	size_t cf_size;
	struct stat cf_stat;
	union {
		int fd;
		DIR *dir;
	} cf_u;
};

static int
cb_open(UNUSED void *arg, const char *filename, void **hp)
{
	struct stat st;
	struct cb_file *cf;
	char path[PATH_MAX];

	if (!host_base)
		return (ENOENT);

	strlcpy(path, host_base, PATH_MAX);
	if (path[strlen(path) - 1] == '/')
		path[strlen(path) - 1] = 0;
	strlcat(path, filename, PATH_MAX);
	cf = malloc(sizeof(struct cb_file));
	if (stat(path, &cf->cf_stat) < 0) {
		free(cf);
		return (errno);
	}

	cf->cf_size = (size_t) st.st_size;
	if (S_ISDIR(cf->cf_stat.st_mode)) {
		cf->cf_isdir = 1;
		cf->cf_u.dir = opendir(path);
		if (!cf->cf_u.dir)
			goto out;
		*hp = cf;
		return (0);
	}
	if (S_ISREG(cf->cf_stat.st_mode)) {
		cf->cf_isdir = 0;
		cf->cf_u.fd = open(path, O_RDONLY);
		if (cf->cf_u.fd < 0)
			goto out;
		*hp = cf;
		return (0);
	}

out:
	free(cf);
	return (EINVAL);
}

// static int
// cb_close(UNUSED void *arg, void *h)
// {
// 	struct cb_file *cf = h;
//
// 	if (cf->cf_isdir)
// 		closedir(cf->cf_u.dir);
// 	else
// 		close(cf->cf_u.fd);
// 	free(cf);
//
// 	return (0);
// }

// static int
// cb_isdir(UNUSED void *arg, void *h)
// {
// 	struct cb_file *cf = h;
//
// 	return (cf->cf_isdir);
// }

// static int
// cb_read(UNUSED void *arg, void *h, void *buf, size_t size, size_t *resid)
// {
// 	struct cb_file *cf = h;
// 	ssize_t sz;
//
// 	if (cf->cf_isdir)
// 		return (EINVAL);
// 	sz = read(cf->cf_u.fd, buf, size);
// 	if (sz < 0)
// 		return (EINVAL);
// 	*resid = size - ((size_t) sz);
// 	return (0);
// }

//static int
//cb_readdir(UNUSED void *arg, void *h, uint32_t *fileno_return,
//	uint8_t *type_return, size_t *namelen_return, char *name)
//{
//	struct cb_file *cf = h;
//	struct dirent *dp;
//
//	if (!cf->cf_isdir)
//		return (EINVAL);
//
//	dp = readdir(cf->cf_u.dir);
//	if (!dp)
//		return (ENOENT);
//
//	/*
//	 * Note: d_namlen is in the range 0..255 and therefore less
//	 * than PATH_MAX so we don't need to test before copying.
//	 */
//	*fileno_return = dp->d_fileno;
//	*type_return = dp->d_type;
//	*namelen_return = dp->d_namlen;
//	memcpy(name, dp->d_name, dp->d_namlen);
//	name[dp->d_namlen] = 0;
//
//	return (0);
//}

// static int
// cb_seek(UNUSED void *arg, void *h, uint64_t offset, int whence)
// {
// 	struct cb_file *cf = h;
//
// 	if (cf->cf_isdir)
// 		return (EINVAL);
// 	if (lseek(cf->cf_u.fd, ((off_t) offset), whence) < 0)
// 		return (errno);
// 	return (0);
// }

// static int
// cb_stat(UNUSED void *arg, void *h, int *mode, int *uid, int *gid,
// 	uint64_t *size)
// {
// 	struct cb_file *cf = h;
//
// 	*mode = cf->cf_stat.st_mode;
// 	*uid = (int) cf->cf_stat.st_uid;
// 	*gid = (int) cf->cf_stat.st_gid;
// 	*size = (uint64_t) cf->cf_stat.st_size;
// 	return (0);
// }

/*
 * Disk image i/o callbacks
 */

static int
cb_diskread(UNUSED void *arg, int unit, uint64_t from, void *to, size_t size,
	size_t *resid)
{
	ssize_t n;

	if (unit < 0 || unit >= ndisks )
		return (EIO);
	n = pread(disk_fd[unit], to, size, ((off_t) from));
	if (n < 0)
		return (errno);
	*resid = size - ((size_t) n);
	return (0);
}

#define DIOCGSECTORSIZE _IOR('d', 128, u_int)
#define DIOCGMEDIASIZE  _IOR('d', 129, off_t)

static int
cb_diskioctl(UNUSED void *arg, int unit, u_long cmd, void *data)
{
	struct stat sb;

	if (unit < 0 || unit >= ndisks)
		return (EBADF);

	switch (cmd) {
	case DIOCGSECTORSIZE:
		*(u_int *)data = 512;
		break;
	case DIOCGMEDIASIZE:
		if (fstat(disk_fd[unit], &sb) == 0)
			*(off_t *)data = sb.st_size;
		else
			return (ENOTTY);
		break;
	default:
		abort();
		return (ENOTTY);
	}

	return (0);
}

/*
 * Guest virtual machine i/o callbacks
 */
static int
cb_copyin(UNUSED void *arg, const void *from, uint64_t to, size_t size)
{
	char *ptr;

	to &= 0x7fffffff;

	ptr = xh_vm_map_gpa(to, size);
	if (ptr == NULL)
		return (EFAULT);

	memcpy(ptr, from, size);
	return (0);
}

static int
cb_copyout(UNUSED void *arg, uint64_t from, void *to, size_t size)
{
	char *ptr;

	from &= 0x7fffffff;

	ptr = xh_vm_map_gpa(from, size);
	if (ptr == NULL)
		return (EFAULT);

	memcpy(to, ptr, size);
	return (0);
}

static void
cb_setreg(UNUSED void *arg, int r, uint64_t v)
{
	int error;
	enum vm_reg_name vmreg;

	vmreg = VM_REG_LAST;

	switch (r) {
	case 4:
		vmreg = VM_REG_GUEST_RSP;
		vcpu_rsp = v;
		break;
	default:
		break;
	}

	if (vmreg == VM_REG_LAST) {
		abort();
	}

	error = xh_vm_set_register(BSP, vmreg, v);
	if (error) {
		perror("xh_vm_set_register");
		cb_exit();
	}
}

static void
cb_setmsr(UNUSED void *arg, int r, uint64_t v)
{
	int error;
	enum vm_reg_name vmreg;

	vmreg = VM_REG_LAST;

	switch (r) {
	case MSR_EFER:
		vmreg = VM_REG_GUEST_EFER;
		break;
	default:
		break;
	}

	if (vmreg == VM_REG_LAST) {
		abort();
	}

	error = xh_vm_set_register(BSP, vmreg, v);
	if (error) {
		perror("xh_vm_set_msr");
		cb_exit();
	}
}

static void
cb_setcr(UNUSED void *arg, int r, uint64_t v)
{
	int error;
	enum vm_reg_name vmreg;

	vmreg = VM_REG_LAST;

	switch (r) {
	case 0:
		vmreg = VM_REG_GUEST_CR0;
		break;
	case 3:
		vmreg = VM_REG_GUEST_CR3;
		vcpu_cr3 = v;
		break;
	case 4:
		vmreg = VM_REG_GUEST_CR4;
		break;
	default:
		break;
	}

	if (vmreg == VM_REG_LAST) {
		fprintf(stderr, "test_setcr(%d): not implemented\n", r);
		cb_exit();
	}

	error = xh_vm_set_register(BSP, vmreg, v);
	if (error) {
		perror("vm_set_cr");
		cb_exit();
	}
}

static void
cb_setgdt(UNUSED void *arg, uint64_t base, size_t size)
{
	int error;

	error = xh_vm_set_desc(BSP, VM_REG_GUEST_GDTR, base,
		((uint32_t) (size - 1)), 0);

	if (error != 0) {
		perror("vm_set_desc(gdt)");
		cb_exit();
	}

	vcpu_gdt_base = base;
}

__attribute__ ((noreturn)) static void
cb_exec(UNUSED void *arg, uint64_t rip)
{
	int error;

	if (vcpu_cr3 == 0) {
		error = fbsd_set_regs_i386(((uint32_t) rip), ((uint32_t) vcpu_gdt_base),
			((uint32_t) vcpu_rsp));
	} else {
		error = fbsd_set_regs(rip, vcpu_cr3, vcpu_gdt_base, vcpu_rsp);
	}

	if (error) {
		perror("fbsd_set_regs");
		cb_exit();
	}

	vcpu_rip = rip;

	longjmp(exec_done, 1);
}

/*
 * Misc
 */

static void
cb_delay(UNUSED void *arg, int usec)
{
	usleep((useconds_t) usec);
}

__attribute__ ((noreturn)) static void
cb_exit(void)
{
	tcsetattr(consout_fd, TCSAFLUSH, &oldterm);
	fprintf(stderr, "fbsd: error\n");
	exit(1);
}

static void
cb_getmem(UNUSED void *arg, uint64_t *ret_lowmem, uint64_t *ret_highmem)
{
	*ret_lowmem = xh_vm_get_lowmem_size();
	*ret_highmem = xh_vm_get_highmem_size();
}

struct env {
	const char *str; /* name=value */
	SLIST_ENTRY(env) next;
};

static SLIST_HEAD(envhead, env) envhead;

static void
addenv(const char *str)
{
	struct env *env;

	env = malloc(sizeof(struct env));
	env->str = str;
	SLIST_INSERT_HEAD(&envhead, env, next);
}

static const char *
cb_getenv(UNUSED void *arg, int num)
{
	int i;
	struct env *env;

	i = 0;
	SLIST_FOREACH(env, &envhead, next) {
		if (i == num)
			return (env->str);
		i++;
	}

	return (NULL);
}

static struct loader_callbacks cb = {
	.getc = cb_getc,
	.putc = cb_putc,
	.poll = cb_poll,

	// .open = cb_open,
	// .close = cb_close,
	// .isdir = cb_isdir,
	// .read = cb_read,
	// .readdir = cb_readdir,
	// .seek = cb_seek,
	// .stat = cb_stat,

	.open = cb_open,
	.close = NULL,
	.isdir = NULL,
	.read = NULL,
	.readdir = NULL,
	.seek = NULL,
	.stat = NULL,

	.diskread = cb_diskread,
	.diskioctl = cb_diskioctl,

	.copyin = cb_copyin,
	.copyout = cb_copyout,
	.setreg = cb_setreg,
	.setmsr = cb_setmsr,
	.setcr = cb_setcr,
	.setgdt = cb_setgdt,
	.exec = cb_exec,

	.delay = cb_delay,
	.exit = cb_exit,
	.getmem = cb_getmem,

	.getenv = cb_getenv,
};

static int
altcons_open(char *path)
{
	struct stat sb;
	int err;
	int fd;

	/*
	 * Allow stdio to be passed in so that the same string
	 * can be used for the bhyveload console and bhyve com-port
	 * parameters
	 */
	if (!strcmp(path, "stdio"))
		return (0);

	err = stat(path, &sb);
	if (err == 0) {
		if (!S_ISCHR(sb.st_mode))
			err = ENOTSUP;
		else {
			fd = open(path, O_RDWR | O_NONBLOCK);
			if (fd < 0)
				err = errno;
			else
				consin_fd = consout_fd = fd;
		}
	}

	return (err);
}

static int
disk_open(char *path)
{
	int err, fd;

	if (ndisks >= NDISKS)
		return (ERANGE);

	err = 0;
	fd = open(path, O_RDONLY);

	if (fd > 0) {
		disk_fd[ndisks] = fd;
		ndisks++;
	} else
		err = errno;

	return (err);
}

int
fbsd_init(char *userboot_path, char *bootvolume_path, char *kernelenv,
	char *cons)
{
	if (!userboot_path || !bootvolume_path)
		return 1;

	config.userboot = userboot_path;
	config.bootvolume = bootvolume_path;
	config.kernelenv = kernelenv;
	config.cons = cons;

	return 0;
}

uint64_t
fbsd_load(void)
{
	void *h;
	int i;
	func_t func;

	host_base = NULL;
	consin_fd = STDIN_FILENO;
	consout_fd = STDOUT_FILENO;

	if (config.cons) {
		altcons_open(config.cons);
	}

	disk_open(config.bootvolume);

	if (config.kernelenv) {
		addenv(config.kernelenv);
	}

	//host_base = optarg h

	tcgetattr(consout_fd, &term);
	oldterm = term;
	cfmakeraw(&term);
	term.c_cflag |= CLOCAL;

	tcsetattr(consout_fd, TCSAFLUSH, &term);

	h = dlopen(config.userboot, RTLD_LOCAL);
	if (!h) {
		fprintf(stderr, "%s\n", dlerror());
		exit(1);
	}

	func = (func_t) dlsym(h, "loader_main");
	if (!func) {
		fprintf(stderr, "%s\n", dlerror());
		exit(1);
	}

	addenv("smbios.bios.vendor=BHYVE");
	addenv("boot_serial=1");

	if (!setjmp(exec_done)) {
		func(&cb, NULL, USERBOOT_VERSION_3, ndisks);
	}

	for (i = 0; i < ndisks; i++) {
		close(disk_fd[i]);
	}

	if (config.cons) {
		assert(consin_fd == consout_fd);
		close(consin_fd);
	}

	return vcpu_rip;
}
