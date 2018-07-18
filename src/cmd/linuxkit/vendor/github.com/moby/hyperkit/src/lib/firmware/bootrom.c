/*-
 * Copyright (c) 2015 Neel Natu <neel@freebsd.org>
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
 * THIS SOFTWARE IS PROVIDED BY THE AUTHOR ``AS IS'' AND
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

#include <sys/param.h>

#include <sys/types.h>
#include <sys/mman.h>
#include <sys/stat.h>

#include <xhyve/vmm/vmm.h>
#include <xhyve/vmm/vmm_mem.h>
#include <xhyve/vmm/vmm_api.h>
#include <xhyve/firmware/bootrom.h>

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <inttypes.h>
#include <stdbool.h>

#define	MAX_BOOTROM_SIZE	(16 * 1024 * 1024)	/* 16 MB */

static const char *romfile;
static uint64_t bootrom_gpa = (1ULL << 32);

int
bootrom_init(const char *romfile_path)
{
	if (!romfile_path)
		return 1;

	romfile = romfile_path;

	return 0;
}

const char * bootrom(void)
{
	return romfile;
}

uint64_t bootrom_load(void)
{

	struct stat sbuf;
	uint64_t gpa;
	ssize_t rlen;
	char *ptr;
	int fd, i, rv;

	rv = -1;
	fd = open(romfile, O_RDONLY);
	if (fd < 0) {
		fprintf(stderr, "Error opening bootrom \"%s\": %s\n",
		    romfile, strerror(errno));
		goto done;
	}

        if (fstat(fd, &sbuf) < 0) {
		fprintf(stderr, "Could not fstat bootrom file \"%s\": %s\n",
		    romfile, strerror(errno));
		goto done;
        }

	/*
	 * Limit bootrom size to 16MB so it doesn't encroach into reserved
	 * MMIO space (e.g. APIC, HPET, MSI).
	 */
	if (sbuf.st_size > MAX_BOOTROM_SIZE || sbuf.st_size < XHYVE_PAGE_SIZE) {
		fprintf(stderr, "Invalid bootrom size %lld\n", (long long)sbuf.st_size);
		goto done;
	}

	if (sbuf.st_size & XHYVE_PAGE_MASK) {
		fprintf(stderr, "Bootrom size %lld is not a multiple of the "
		    "page size\n", (long long)sbuf.st_size);
		goto done;
	}

	gpa = bootrom_gpa -= (size_t)sbuf.st_size;

	/* XXX Mapping cold be R/O to guest */
	ptr = vmm_mem_alloc(gpa, (size_t)sbuf.st_size);
	if (!ptr) {
		fprintf(stderr,
			"Failed to allocate %lld bytes of memory for bootrom\n",
			(long long)sbuf.st_size);
		rv = -1;
		goto done;
	}

	/* Read 'romfile' into the guest address space */
	for (i = 0; i < sbuf.st_size / XHYVE_PAGE_SIZE; i++) {
		rlen = read(fd, ptr + i * XHYVE_PAGE_SIZE, XHYVE_PAGE_SIZE);
		if (rlen != XHYVE_PAGE_SIZE) {
			fprintf(stderr, "Incomplete read of page %d of bootrom "
			    "file %s: %ld bytes\n", i, romfile, rlen);
			goto done;
		}
	}

	rv = 0;
done:
	if (fd >= 0)
		close(fd);
	if (rv)
		exit(1);
	return 0xfff0;
}

bool
bootrom_contains_gpa(uint64_t gpa)
{
	return (gpa >= bootrom_gpa && gpa < (1ULL << 32));
}
