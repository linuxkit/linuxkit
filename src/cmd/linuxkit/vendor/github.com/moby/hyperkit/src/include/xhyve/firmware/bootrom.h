#pragma once

#include <stdint.h>
#include <stdbool.h>

int bootrom_init(const char *bootrom_path);
const char *bootrom(void);
uint64_t bootrom_load(void);
bool bootrom_contains_gpa(uint64_t gpa);
