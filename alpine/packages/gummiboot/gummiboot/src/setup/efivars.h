/*-*- Mode: C; c-basic-offset: 8; indent-tabs-mode: nil -*-*/

#pragma once

/***
  This file is part of systemd.

  Copyright 2013 Lennart Poettering

  systemd is free software; you can redistribute it and/or modify it
  under the terms of the GNU Lesser General Public License as published by
  the Free Software Foundation; either version 2.1 of the License, or
  (at your option) any later version.

  systemd is distributed in the hope that it will be useful, but
  WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
  Lesser General Public License for more details.

  You should have received a copy of the GNU Lesser General Public License
  along with systemd; If not, see <http://www.gnu.org/licenses/>.
***/

#include <stdbool.h>
#include <sys/types.h>
#include <inttypes.h>

#define EFI_VARIABLE_NON_VOLATILE       0x0000000000000001
#define EFI_VARIABLE_BOOTSERVICE_ACCESS 0x0000000000000002
#define EFI_VARIABLE_RUNTIME_ACCESS     0x0000000000000004

#define EFI_VENDOR_GLOBAL ((uint8_t[16]) { 0x8b,0xe4,0xdf,0x61,0x93,0xca,0x11,0xd2,0xaa,0x0d,0x00,0xe0,0x98,0x03,0x2b,0x8c })
#define EFI_VENDOR_LOADER ((uint8_t[16]) { 0x4a,0x67,0xb0,0x82,0x0a,0x4c,0x41,0xcf,0xb6,0xc7,0x44,0x0b,0x29,0xbb,0x8c,0x4f })

bool is_efi_boot(void);
int is_efi_secure_boot(void);
int is_efi_secure_boot_setup_mode(void);
int efi_get_variable(const uint8_t vendor[16], const char *name, void **value, size_t *size);
int efi_set_variable( const uint8_t vendor[16], const char *name, const void *value, size_t size);
int efi_get_variable_string(const uint8_t vendor[16], const char *name, char **p);
int efi_get_boot_option(uint16_t id, char **title, uint8_t part_uuid[16], char **path, bool *active);

int efi_get_boot_options(uint16_t **options);
int efi_add_boot_option(uint16_t id, const char *title,
                        uint32_t part, uint64_t pstart, uint64_t psize,
                        const uint8_t part_uuid[16],
                        const char *path);
int efi_remove_boot_option(uint16_t id);

int efi_get_boot_order(uint16_t **order);
int efi_set_boot_order(uint16_t *order, size_t n);

char *utf16_to_utf8(const void *s, size_t length);
char *tilt_backslashes(char *s);
uint16_t *tilt_slashes(uint16_t *s);
