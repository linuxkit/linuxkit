#include <stdint.h>

int multiboot_init(char *kernel_path, char *module_spec, char *cmdline);
uint64_t multiboot(void);
