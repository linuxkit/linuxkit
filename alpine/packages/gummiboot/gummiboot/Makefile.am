#
#  This file is part of gummiboot
#
#  Copyright (C) 2013 Karel Zak <kzak@redhat.com>
#
#  gummiboot is free software; you can redistribute it and/or modify it
#  under the terms of the GNU Lesser General Public License as published by
#  the Free Software Foundation; either version 2.1 of the License, or
#  (at your option) any later version.
#
#  gummiboot is distributed in the hope that it will be useful, but
#  WITHOUT ANY WARRANTY; without even the implied warranty of
#  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
#  Lesser General Public License for more details.
#
#  You should have received a copy of the GNU Lesser General Public License
#  along with systemd; If not, see <http://www.gnu.org/licenses/>.

ACLOCAL_AMFLAGS = -I m4 ${ACLOCAL_FLAGS}
AM_MAKEFLAGS = --no-print-directory

gummibootlibdir = $(prefix)/lib/gummiboot

AM_CPPFLAGS = -include config.h
AM_CFLAGS = \
	-D_GNU_SOURCE \
	-Wall \
	-Wextra \
	-Wmissing-prototypes \
	-Wno-unused-parameter
AM_LDFLAGS =

EXTRA_DIST = autogen.sh README LICENSE
CLEANFILES =

# ------------------------------------------------------------------------------
if HAVE_BLKID
bin_PROGRAMS = gummiboot

gummiboot_SOURCES = \
	src/setup/setup.c \
	src/setup/efivars.c \
	src/setup/efivars.h

gummiboot_CPPFLAGS = \
	$(AM_CPPFLAGS) \
	-DMACHINE_TYPE_NAME=\"$(MACHINE_TYPE_NAME)\" \
	-DGUMMIBOOTLIBDIR=\"$(gummibootlibdir)\"

gummiboot_CFLAGS = \
	$(AM_CFLAGS) \
	$(BLKID_CFLAGS)

gummiboot_LDADD = \
	$(BLKID_LIBS)
endif

if ENABLE_MANPAGES
%.8: %.xml
	$(AM_V_GEN)$(XSLTPROC) -o $@ --nonet \
	  --stringparam man.output.quietly 1 \
	  --stringparam man.th.extra1.suppress 1 \
	  --stringparam man.authors.section.enabled 0 \
	  --stringparam man.copyright.section.enabled 0 \
	  http://docbook.sourceforge.net/release/xsl/current/manpages/docbook.xsl $<

dist_man_MANS = man/gummiboot.8
endif

EXTRA_DIST += man/gummiboot.xml
CLEANFILES += man/gummiboot.8

# ------------------------------------------------------------------------------
# EFI compilation -- this part of the build system uses custom make rules and
# bypasses regular automake to provide absolute control on compiler and linker
# flags.
efi_cppflags = \
	$(EFI_CPPFLAGS) \
	-I$(top_builddir) -include config.h \
	-I$(EFI_INC_DIR)/efi \
	-I$(EFI_INC_DIR)/efi/$(ARCH) \
	-DMACHINE_TYPE_NAME=\"$(MACHINE_TYPE_NAME)\"

efi_cflags = \
	$(EFI_CFLAGS) \
	-Wall \
	-Wextra \
	-std=gnu90 \
	-nostdinc \
	-ggdb -O0 \
	-fpic \
	-fshort-wchar \
	-nostdinc \
	-ffreestanding \
	-fno-strict-aliasing \
	-fno-stack-protector \
	-Wsign-compare \
	-mno-sse \
	-mno-mmx

if ARCH_X86_64
efi_cflags += \
	-mno-red-zone \
	-DEFI_FUNCTION_WRAPPER \
	-DGNU_EFI_USE_MS_ABI
endif

efi_ldflags = \
	$(EFI_LDFLAGS) \
	-T $(EFI_LDS_DIR)/elf_$(ARCH)_efi.lds \
	-shared \
	-Bsymbolic \
	-nostdlib \
	-znocombreloc \
	-L $(EFI_LIB_DIR) \
	$(EFI_LDS_DIR)/crt0-efi-$(ARCH).o

# ------------------------------------------------------------------------------
gummiboot_headers = \
	src/efi/util.h \
	src/efi/console.h \
	src/efi/graphics.h \
	src/efi/pefile.h

gummiboot_sources = \
	src/efi/util.c \
	src/efi/console.c \
	src/efi/graphics.c \
	src/efi/pefile.c \
	src/efi/gummiboot.c

gummiboot_objects = $(addprefix $(top_builddir)/,$(gummiboot_sources:.c=.o))
gummiboot_solib = $(top_builddir)/src/efi/gummiboot.so
gummiboot = gummiboot$(MACHINE_TYPE_NAME).efi

gummibootlib_DATA = $(gummiboot)
CLEANFILES += $(gummiboot_objects) $(gummiboot_solib) $(gummiboot)
EXTRA_DIST += $(gummiboot_sources) $(gummiboot_headers)

$(top_builddir)/src/efi/%.o: $(top_srcdir)/src/efi/%.c $(addprefix $(top_srcdir)/,$(gummiboot_headers))
	@$(MKDIR_P) $(top_builddir)/src/efi/
	$(AM_V_CC)$(EFI_CC) $(efi_cppflags) $(efi_cflags) -c $< -o $@

$(gummiboot_solib): $(gummiboot_objects)
	$(AM_V_CCLD)$(LD) $(efi_ldflags) $(gummiboot_objects) \
		-o $@ -lefi -lgnuefi $(shell $(CC) -print-libgcc-file-name); \
	nm -D -u $@ | grep ' U ' && exit 1 || :
.DELETE_ON_ERROR: $(gummboot_solib)

$(gummiboot): $(gummiboot_solib)
	$(AM_V_GEN) objcopy -j .text -j .sdata -j .data -j .dynamic \
	  -j .dynsym -j .rel -j .rela -j .reloc \
	  --target=efi-app-$(ARCH) $< $@

# ------------------------------------------------------------------------------
stub_headers = \
	src/efi/util.h \
	src/efi/pefile.h \
	src/efi/linux.h

stub_sources = \
	src/efi/util.c \
	src/efi/pefile.c \
	src/efi/linux.c \
	src/efi/stub.c

stub_objects = $(addprefix $(top_builddir)/,$(stub_sources:.c=.o))
stub_solib = $(top_builddir)/src/efi/stub.so
stub = linux$(MACHINE_TYPE_NAME).efi.stub

gummibootlib_DATA += $(stub)
CLEANFILES += $(stub_objects) $(stub_solib) $(stub)
EXTRA_DIST += $(stub_sources) $(stub_headers)

$(top_builddir)/src/efi/%.o: $(top_srcdir)/src/efi/%.c $(addprefix $(top_srcdir)/,$(stub_headers))
	@$(MKDIR_P) $(top_builddir)/src/efi/
	$(AM_V_CC)$(EFI_CC) $(efi_cppflags) $(efi_cflags) -c $< -o $@

$(stub_solib): $(stub_objects)
	$(AM_V_CCLD)$(LD) $(efi_ldflags) $(stub_objects) \
		-o $@ -lefi -lgnuefi $(shell $(CC) -print-libgcc-file-name); \
	nm -D -u $@ | grep ' U ' && exit 1 || :
.DELETE_ON_ERROR: $(gummboot_solib)

$(stub): $(stub_solib)
	$(AM_V_GEN) objcopy -j .text -j .sdata -j .data -j .dynamic \
	  -j .dynsym -j .rel -j .rela -j .reloc \
	  --target=efi-app-$(ARCH) $< $@

# ------------------------------------------------------------------------------
CLEANFILES += test-disk.img
EXTRA_DIST += test/test-create-disk.sh

test-disk.img: gummiboot$(MACHINE_TYPE_NAME).efi test/test-create-disk.sh
	$(AM_V_GEN)test/test-create-disk.sh

qemu: test-disk.img
	$(QEMU) -machine accel=kvm -m 1024 -bios $(QEMU_BIOS) -snapshot test-disk.img

install-tree: all
	rm -rf $(abs_srcdir)/install-tree
	$(MAKE) install DESTDIR=$(abs_srcdir)/install-tree
	tree $(abs_srcdir)/install-tree
