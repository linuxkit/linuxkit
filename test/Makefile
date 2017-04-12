.PHONY: default all
default: test-kernel-config
all: default

MOBY?=../bin/moby

define check_test_log
	@cat $1 |grep -q 'Moby test suite PASSED'
endef

test-kernel-config-initrd.img: cases/test-kernel-config.yml
	$(MOBY) build cases/test-kernel-config.yml

.PHONY: test-kernel-config
test-kernel-config: test-kernel-config-initrd.img test-kernel-config-bzImage test-kernel-config-cmdline
	rm -f disk.img
	script -q /dev/null $(MOBY) run test-kernel-config | tee test.log
	$(call check_test_log, test.log)

# interactive versions need to use volume mounts
.PHONY: test-qemu-efi
test-qemu-efi: test-kernel-config-efi.iso
	script -q /dev/null $(MOBY) run -uefi test-kernel-config tee test-efi.log
	$(call check_test_log, test-efi.log)

.PHONY: test-gcp
test-gcp: $(MOBY) test.img.tar.gz
	script -q /dev/null $(MOBY) run gcp test-kernel-config.img.tar.gz | tee test-gcp.log
	$(call check_test_log, test-gcp.log)

.PHONY: clean
clean:
	rm -rf bin *.log *-bzImage *-cmdline *.img *.iso *.tar.gz *.qcow2 *.vhd
