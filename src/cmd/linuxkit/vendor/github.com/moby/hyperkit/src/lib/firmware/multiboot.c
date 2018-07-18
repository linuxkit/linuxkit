/*-
 * Copyright (c) 2016 Thomas Haggett
 * Copyright (c) 2017 Pavel Borzenkov
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
 * THIS SOFTWARE IS PROVIDED BY THOMAS HAGGETT ``AS IS'' AND
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
 */

#include <stdint.h>
#include <stdlib.h>
#include <stdio.h>
#include <stdarg.h>
#include <string.h>
#include <errno.h>
#include <unistd.h>

#include <sys/stat.h>
#include <sys/fcntl.h>
#include <sys/mman.h>

#include <xhyve/firmware/multiboot.h>
#include <xhyve/vmm/vmm_api.h>

#define MULTIBOOT_MAGIC 0x1BADB002
#define MULTIBOOT_SEARCH_END 0x2000

struct multiboot_load_header {
	uint32_t header_addr;
	uint32_t load_addr;
	uint32_t load_end_addr;
	uint32_t bss_end_addr;
	uint32_t entry_addr;
};

struct multiboot_video_header {
	uint32_t mode_type;
	uint32_t width;
	uint32_t height;
	uint32_t depth;
};

struct multiboot_header {
	struct {
		uint32_t magic;
		uint32_t flags;
		uint32_t checksum;
	} hdr;
	struct multiboot_load_header lhdr;
	struct multiboot_video_header vhdr;
};

struct multiboot_info  {
	uint32_t flags;
	uint32_t mem_lower;
	uint32_t mem_upper;
	uint32_t boot_device;
	uint32_t cmdline_addr;
	uint32_t mods_count;
	uint32_t mods_addr;
};

struct multiboot_module_entry {
	uint32_t addr_start;
	uint32_t addr_end;
	uint32_t cmdline;
	uint32_t pad;
};

static struct multiboot_config {
	char* kernel_path;
	char* module_list;
	char* kernel_append;
} config;

struct image {
	void *mapping;
	uint32_t size;
};

struct multiboot_state {
	uintptr_t guest_mem_base; // HVA of guest's memory
	uint32_t guest_mem_size; // size of guest's memory

	uint32_t load_addr; // current data load GPA

	uint32_t kernel_load_addr; // kernel load GPA
	uint32_t kernel_size; // size of the kernel
	uint32_t kernel_offset; // offset of the kernel in file
	uint32_t kernel_entry_addr; // kernel entry GPA

	uint32_t mbi_addr; // GPA of MBI header

	struct multiboot_info mbi; // MBI
	struct multiboot_module_entry *modules; // modules info
	unsigned modules_count;

	char *cmdline; // combined kernel and modules cmdline
	uint32_t cmdline_len;
};

struct elf_ehdr {
	uint8_t e_ident[16];
	uint16_t e_type;
	uint16_t e_machine;
	uint32_t e_version;
	uint32_t e_entry;
	uint32_t e_phoff;
	uint32_t e_shoff;
	uint32_t e_flags;
	uint16_t e_hsize;
	uint16_t e_phentsize;
	uint16_t e_phnum;
	uint16_t e_shentsize;
	uint16_t e_shnum;
	uint16_t e_shstrndx;
};

#define EM_X86_64 62

struct elf_phdr {
	uint32_t p_type;
	uint32_t p_offset;
	uint32_t p_vaddr;
	uint32_t p_paddr;
	uint32_t p_filesz;
	uint32_t p_memsz;
	uint32_t p_flags;
	uint32_t p_align;
};

#define PT_LOAD 1

#define PF_X 0x1

#define ROUND_UP(a, b) (((a) + (b) - 1) / (b) * (b))
#define MIN(a, b) ((a) < (b) ? (a) : (b))

#define PAGE_SIZE 4096

static void __attribute__((noreturn,format(printf,1,2)))
die(const char *format, ...)
{
	va_list args;
	va_start(args, format);
	vfprintf(stderr, format, args);
	va_end(args);

	exit(1);
}

// get_image maps file at path into process's address space.
static void
get_image(char *path, struct image *img)
{
	int fd;
	struct stat st;

	fd = open(path, O_RDONLY);
	if (fd < 0)
		die("multiboot: failed to open '%s': %s\n", path, strerror(errno));
	if (fstat(fd, &st) < 0) {
		close(fd);
		die("multiboot: failed to stat '%s': %s\n", path, strerror(errno));
	}
	img->size = (uint32_t)st.st_size;

	img->mapping = mmap(NULL, ROUND_UP(img->size, 4096), PROT_READ, MAP_PRIVATE, fd, 0);
	close(fd);
	if (img->mapping == (void *)MAP_FAILED)
		die("multiboot: failed to mmap '%s': %s\n", path, strerror(errno));
}

// put_image releases mmaped file.
static int
put_image(struct image *img)
{
	if (munmap(img->mapping, ROUND_UP(img->size, 4096)) < 0)
		die("multiboot: failed to munmap: %s\n", strerror(errno));
	img->mapping = NULL;
	img->size = 0;

	return 0;
}

// multiboot_init is called by xhyve to pass in the firmware arguments.
int
multiboot_init(char *kernel_path, char *module_list, char *kernel_append)
{
	if (!kernel_path)
		return 1;

	config.kernel_path = kernel_path;
	config.module_list = module_list;
	config.kernel_append = kernel_append;

	return 0;
}

// multiboot_parse_elf parses ELF header and determines kernel load/entry
// addresses.
static void
multiboot_parse_elf(struct multiboot_state *s, struct image *img)
{
	struct elf_ehdr *ehdr = img->mapping;
	struct elf_phdr *phdr;
	uint32_t low = (uint32_t)-1, high = 0, memsize, addr, entry;
	int i;

	if (ehdr->e_ident[0] != 0x7f ||
			ehdr->e_ident[1] != 'E' ||
			ehdr->e_ident[2] != 'L' ||
			ehdr->e_ident[3] != 'F')
		die("multiboot: invalid ELF magic\n");
	if (ehdr->e_machine == EM_X86_64)
		die("multiboot: 64-bit ELFs are not supported\n");

	entry = ehdr->e_entry;

	phdr = (struct elf_phdr *)((uintptr_t)img->mapping + ehdr->e_phoff);
	for (i = 0; i < ehdr->e_phnum; i++) {
		if (phdr[i].p_type != PT_LOAD)
			continue;
		memsize = phdr[i].p_filesz;
		addr = phdr[i].p_paddr;

		if (phdr[i].p_flags & PF_X &&
				phdr[i].p_vaddr != phdr[i].p_paddr &&
				entry >= phdr[i].p_vaddr &&
				entry < phdr[i].p_vaddr + phdr[i].p_filesz)
			entry = entry - phdr[i].p_vaddr + phdr[i].p_paddr;

		if (addr < low)
			low = addr;
		if (addr + memsize > high)
			high = addr + memsize;
	}

	if (low == (uint32_t)-1 || high == 0)
		die("multiboot: failed to parse ELF file\n");

	s->kernel_load_addr = low;
	s->kernel_size = high - low;
	s->kernel_entry_addr = entry;
}

// multiboot_find_header scans the configured kernel for it's multiboot header.
static void
multiboot_find_header(struct multiboot_state *s, struct image *img)
{
	struct multiboot_header *header = NULL;
	uintptr_t ptr = (uintptr_t)img->mapping;
	uintptr_t sz = MIN(img->size, MULTIBOOT_SEARCH_END);
	uintptr_t end = ptr + sz - sizeof(struct multiboot_header);
	int found = 0;

	for (; ptr < end; ptr += 4) {
		header = (struct multiboot_header *)ptr;

		if (header->hdr.magic != MULTIBOOT_MAGIC)
			continue;
		if (header->hdr.checksum + header->hdr.flags + header->hdr.magic != 0)
			continue;

		found = 1;
		break;
	}
	if (!found)
		die("multiboot: failed to find multiboot header in '%s'\n", config.kernel_path);

	// are there any mandatory flags that we don't support? (any other than 0 and 1 set)
	uint16_t supported_mandatory = ((1 << 1) | (1 << 0));
	if (((header->hdr.flags & ~supported_mandatory) & 0xFFFF) != 0x0)
		die("multiboot: header has unsupported mandatory flags (0x%x), bailing.\n", header->hdr.flags & 0xFFFF);

	if (!(header->hdr.flags & (1 << 16))) {
		multiboot_parse_elf(s, img);
		return;
	}

	s->kernel_offset = (uint32_t)((uintptr_t)header - (uintptr_t)img->mapping -
		(header->lhdr.header_addr - header->lhdr.load_addr));
	if (header->lhdr.load_end_addr) {
		s->kernel_size = header->lhdr.load_end_addr - header->lhdr.load_addr;
	} else {
		s->kernel_size = img->size - s->kernel_offset;
	}
	s->kernel_load_addr = header->lhdr.load_addr;
	s->kernel_entry_addr = header->lhdr.entry_addr;
}

// guest_to_host translates GPA to HVA.
static uintptr_t
guest_to_host(uint32_t guest_addr, struct multiboot_state *s)
{
	return s->guest_mem_base + guest_addr;
}

// multiboot_load_data loads data pointed to by from of size into current load
// address in guest's memory. The load address is advanced taken align into
// account.
static uint32_t
multiboot_load_data(struct multiboot_state *s, void *from, uint32_t size, unsigned align)
{
	uintptr_t to = guest_to_host(s->load_addr, s);
	uint32_t loaded_at = s->load_addr;

	if (from == NULL || size == 0)
		return 0;

	if ((s->load_addr >= s->guest_mem_size) ||
			((s->load_addr + size) > s->guest_mem_size))
		die("multiboot: %x+%x is beyond guest's memory\n", s->load_addr, size);

	memcpy((void *)to, from, size);

	s->load_addr = s->load_addr + size;
	if (align)
		s->load_addr = ROUND_UP(s->load_addr, align);

	return loaded_at;
}

// multiboot_set_guest_state prepares guest registers/descriptors per Multiboot
// specification.
static uint64_t
multiboot_set_guest_state(struct multiboot_state *s)
{
	xh_vcpu_reset(0);
	xh_vm_set_register(0, VM_REG_GUEST_RAX, 0x2BADB002);
	xh_vm_set_register(0, VM_REG_GUEST_RBX, s->mbi_addr);
	xh_vm_set_register(0, VM_REG_GUEST_RIP, s->kernel_entry_addr);

	xh_vm_set_desc(0, VM_REG_GUEST_CS, 0, 0xffffffff, 0xc09b);
	xh_vm_set_desc(0, VM_REG_GUEST_DS, 0, 0xffffffff, 0xc093);
	xh_vm_set_desc(0, VM_REG_GUEST_ES, 0, 0xffffffff, 0xc093);
	xh_vm_set_desc(0, VM_REG_GUEST_FS, 0, 0xffffffff, 0xc093);
	xh_vm_set_desc(0, VM_REG_GUEST_GS, 0, 0xffffffff, 0xc093);
	xh_vm_set_desc(0, VM_REG_GUEST_SS, 0, 0xffffffff, 0xc093);

	xh_vm_set_register(0, VM_REG_GUEST_CR0, 0x21);

	return s->kernel_entry_addr;
}

// multiboot_add_cmdline add boot_file cmdline into combined command line
// buffer.
static uint32_t
multiboot_add_cmdline(struct multiboot_state *s, char *boot_file, char *cmdline)
{
	uint32_t offset = s->cmdline_len;

	unsigned len = (unsigned)strlen(boot_file) + 1;
	if (cmdline != NULL)
		len += (unsigned)strlen(cmdline) + 1;

	s->cmdline = realloc(s->cmdline, s->cmdline_len + len);
	if (s->cmdline == NULL)
		die("multiboot: failed to allocate memory for command line\n");

	if (cmdline != NULL)
		snprintf(s->cmdline + s->cmdline_len, len, "%s %s", boot_file, cmdline);
	else
		snprintf(s->cmdline + s->cmdline_len, len, "%s", boot_file);
	s->cmdline_len += len;

	return offset;
}

// multiboot_count_modules returns the amount of passed modules.
static unsigned
multiboot_count_modules()
{
	if (config.module_list == NULL || !*config.module_list)
		return 0;

	unsigned count = 1;
	char *p = config.module_list;
	while (*p) {
		if (*p == ':')
			count++;
		p++;
	}
	return count;
}

// multiboot_process_module processes a single module.
static void
multiboot_process_module(struct multiboot_state *s, char *modname, struct multiboot_module_entry *mod_info)
{
	char *cmd = NULL, *d;
	struct image img;
	if ((d = strchr(modname, ';')) != NULL) {
		cmd = d + 1;
		*d = '\0';
	}

	get_image(modname, &img);
	mod_info->addr_start = multiboot_load_data(s, img.mapping, img.size, PAGE_SIZE);
	mod_info->addr_end = mod_info->addr_start + img.size;
	put_image(&img);
	mod_info->cmdline = multiboot_add_cmdline(s, modname, cmd);
}

uint64_t
multiboot(void)
{
	void *gpa = xh_vm_map_gpa(0, xh_vm_get_lowmem_size());
	uint32_t mem_size = (uint32_t)xh_vm_get_lowmem_size();
	struct multiboot_state state = {
		.guest_mem_base = (uintptr_t)gpa,
		.guest_mem_size = mem_size,
	};
	struct image img;

	// mmap kernel and locate multiboot header
	get_image(config.kernel_path, &img);
	multiboot_find_header(&state, &img);
	state.load_addr = state.kernel_load_addr;

	// actually load the image into the guest's memory
	multiboot_load_data(&state, (void *)((uintptr_t)img.mapping + state.kernel_offset), state.kernel_size, PAGE_SIZE);
	put_image(&img);

	// prepare memory for modules info
	state.modules_count = multiboot_count_modules();
	state.modules = malloc(sizeof(*state.modules) * state.modules_count);
	if (state.modules == NULL)
		die("multiboot: failed to allocate memory for modules info info\n");
	memset(state.modules, 0, sizeof(*state.modules) * state.modules_count);

	// load modules and their command lines
	char *modname;
	unsigned i = 0;
	while ((modname = strsep(&config.module_list, ":")) != NULL)
		multiboot_process_module(&state, modname, &state.modules[i++]);

	// load combined command line (all modules + kernel)
	if (config.kernel_append) {
		state.mbi.cmdline_addr = multiboot_add_cmdline(&state, config.kernel_path, config.kernel_append);
		state.mbi.cmdline_addr += state.load_addr;
	}
	for (i = 0; i < state.modules_count; i++)
		state.modules[i].cmdline += state.load_addr;
	multiboot_load_data(&state, state.cmdline, state.cmdline_len, 4);

	// finally load MBI header
	state.mbi_addr = state.load_addr;
	state.mbi.flags = (1 << 0) | (1 << 2) | (1 << 3);
	state.mbi.mem_lower = (uint32_t)640;
	state.mbi.mem_upper = state.guest_mem_size / 1024 - 1024;
	state.mbi.mods_count = state.modules_count;
	state.mbi.mods_addr = state.mbi_addr + sizeof(state.mbi);
	multiboot_load_data(&state, &state.mbi, sizeof(state.mbi), 0);
	multiboot_load_data(&state, state.modules, sizeof(*state.modules) * state.modules_count, 0);

	free(state.modules);
	free(state.cmdline);
	return multiboot_set_guest_state(&state);
}
