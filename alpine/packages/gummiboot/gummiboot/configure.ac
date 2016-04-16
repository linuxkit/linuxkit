#
#  This file is part of gummiboot.
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

AC_INIT([gummiboot],
        [48],
        [systemd-devel@lists.freedesktop.org],
        [gummiboot],
        [http://freedesktop.org/wiki/Software/gummiboot])

AC_CONFIG_SRCDIR([src/setup/setup.c])
AC_CONFIG_MACRO_DIR([m4])
AC_CONFIG_HEADERS([config.h])
AC_CONFIG_AUX_DIR([build-aux])
AC_PREFIX_DEFAULT([/usr])

AM_INIT_AUTOMAKE([foreign 1.11 -Wall -Wno-portability silent-rules tar-pax no-dist-gzip dist-xz subdir-objects])
AM_SILENT_RULES([yes])

AC_CANONICAL_HOST
AC_SYS_LARGEFILE

AC_PROG_CC

dnl Don't try to use things like -std=c99 for efi compilation
EFI_CC=$CC
AC_SUBST([EFI_CC])

AC_PROG_CC_C99
AM_PROG_CC_C_O
AC_PROG_GCC_TRADITIONAL

AC_PROG_MKDIR_P
AC_PATH_PROG([XSLTPROC], [xsltproc])

dnl Define ARCH_<NAME> conditionals
SET_ARCH(IA32, i*86*)
SET_ARCH(X86_64, x86_64*)
SET_ARCH(IA64, ia64*)

ARCH=`echo $host | sed "s/\(-\).*$//"`

AM_COND_IF(ARCH_IA32, [
        ARCH=ia32
        MACHINE_TYPE_NAME=ia32])

AM_COND_IF(ARCH_X86_64, [
        MACHINE_TYPE_NAME=x64])

AC_SUBST([ARCH])
AC_SUBST([MACHINE_TYPE_NAME])

# QEMU and OVMF UEFI firmware
AS_IF([test x"$cross_compiling" = "xyes"], [], [
        AC_PATH_PROG([QEMU], [qemu-system-x86_64])
        AC_CHECK_FILE([/usr/share/qemu/bios-ovmf.bin], [QEMU_BIOS=/usr/share/qemu/bios-ovmf.bin])
        AC_CHECK_FILE([/usr/share/qemu-ovmf/bios.bin], [QEMU_BIOS=/usr/share/qemu-ovmf/bios.bin])
        AC_SUBST([QEMU_BIOS])
])

# ------------------------------------------------------------------------------
dnl GNU EFI doesn't use relative paths: efi.h includes efibind.h which is in
dnl ${ARCH} relative to efi.h. I can't find a way to get AC_CHECK_HEADERS to
dnl add -I/usr/include/efi/${ARCH} to the conftest.c build. So, just test for
dnl efibind.h as the chances of efi.h not existing if it does are very low.
AC_CHECK_HEADER(efi/${ARCH}/efibind.h, [],
        [AC_MSG_ERROR([*** GNU EFI header efibind.h not found])])

efiroot=$(echo $(cd /usr/lib/$(gcc -print-multi-os-directory); pwd))
EFI_LIB_DIR="$efiroot"
AC_ARG_WITH(efi-libdir,
        AS_HELP_STRING([--with-efi-libdir=PATH], [Path to efi lib directory]),
        [EFI_LIB_DIR="$withval"], [EFI_LIB_DIR="$efiroot"]
)
AC_SUBST([EFI_LIB_DIR])

dnl extra objects and linker scripts
AC_ARG_WITH(efi-ldsdir,
        AS_HELP_STRING([--with-efi-ldsdir=PATH], [Path to efi lds directory]),
        [EFI_LDS_DIR="$withval"],
        [
                for EFI_LDS_DIR in "${efiroot}/gnuefi" "${efiroot}"; do
                    for lds in ${EFI_LDS_DIR}/elf_${ARCH}_efi.lds; do
                            test -f ${lds} && break 2
                    done
                done
        ]
)
AC_SUBST([EFI_LDS_DIR])

AC_ARG_WITH(efi-includedir,
        AS_HELP_STRING([--with-efi-includedir=PATH], [Path to efi include directory]),
        [EFI_INC_DIR="$withval"], [EFI_INC_DIR="/usr/include"]
)
AC_SUBST([EFI_INC_DIR])

# ------------------------------------------------------------------------------
AC_ARG_ENABLE(blkid, AS_HELP_STRING([--disable-blkid], [disable blkid support]))
if test "x$enable_blkid" != "xno"; then
        PKG_CHECK_MODULES(BLKID, [ blkid >= 2.20 ],
                [AC_DEFINE(HAVE_BLKID, 1, [Define if blkid is available]) have_blkid=yes], have_blkid=no)
        if test "x$have_blkid" = xno -a "x$enable_blkid" = xyes; then
                AC_MSG_ERROR([*** blkid support requested but libraries not found])
        fi
fi
AM_CONDITIONAL(HAVE_BLKID, [test "$have_blkid" = "yes"])

# ------------------------------------------------------------------------------
have_manpages=no
AC_ARG_ENABLE(manpages, AS_HELP_STRING([--disable-manpages], [disable manpages]))
AS_IF([test "x$enable_manpages" != xno], [
        AS_IF([test "x$enable_manpages" = xyes -a "x$XSLTPROC" = x], [
                AC_MSG_ERROR([*** Manpages requested but xsltproc not found])
        ])
        AS_IF([test "x$XSLTPROC" != x], [have_manpages=yes])
])
AM_CONDITIONAL(ENABLE_MANPAGES, [test "x$have_manpages" = "xyes"])

# ------------------------------------------------------------------------------
AC_CONFIG_FILES([
        Makefile
])

AC_OUTPUT
AC_MSG_RESULT([
        $PACKAGE_NAME $VERSION

        prefix:                  ${prefix}
        arch:                    $ARCH
        EFI machine type:        $MACHINE_TYPE_NAME

        EFI libdir:              ${EFI_LIB_DIR}
        EFI ldsdir:              ${EFI_LDS_DIR}
        EFI includedir:          ${EFI_INC_DIR}

        blkid:                   ${have_blkid}
        man pages:               ${have_manpages}

        QEMU:                    ${QEMU}
        QEMU OVMF:               ${QEMU_BIOS}
])
