.PHONY: default pr all

default: check-deps kernel-config
pr: check-deps kernel-config ../artifacts/test.img.tar.gz
all: check-deps kernel-config ltp ../artifacts/test.img.tar.gz ../artifacts/test-ltp.img.tar.gz

MOBY:=$(shell command -v moby 2> /dev/null)
LINUXKIT:=$(shell command -v linuxkit 2> /dev/null)

.PHONY: check-deps
check-deps:
ifndef MOBY
	$(error "moby is not available. please install it.")
endif
ifndef LINUXKIT
	$(error "linuxkit binary not found. please install it.")
endif

define check_test_log
	@cat $1 |grep -q 'test suite PASSED'
endef

../artifacts:
	mkdir -p $@

test-kernel-config-initrd.img: $(MOBY) cases/test-kernel-config.yml
	$(MOBY) build --pull cases/test-kernel-config.yml

# uses qemu
.PHONY: kernel-config
kernel-config: $(LINUXKIT) test-kernel-config-initrd.img test-kernel-config-kernel test-kernel-config-cmdline
	$(LINUXKIT) run qemu test-kernel-config 2>&1 | tee test.log
	$(call check_test_log, test.log)

# qemu only at this time
.PHONY: qemu-efi
efi: $(LINUXKIT) test-kernel-config-efi.iso
	$(LINUXKIT) run qemu -uefi test-kernel-config | tee test-efi.log
	$(call check_test_log, test-efi.log)

# For GCP tests
../artifacts/test.img.tar.gz: test-kernel-config.img.tar.gz | ../artifacts
	mv test-kernel-config.img.tar.gz ../artifacts/test.img.tar.gz

.PHONY: gcp
gcp: export CLOUDSDK_IMAGE_NAME?=test-kernel-config
gcp: $(LINUXKIT) ../test.img.tar.gz
	$(LINUXKIT) push gcp ../artifacts/test.img.tar.gz
	$(LINUXKIT) run gcp $(CLOUDSDK_IMAGE_NAME) | tee test-gcp.log
	$(call check_test_log, test-gcp.log)

# For LTP tests
test-ltp.img.tar.gz: $(MOBY) cases/test-ltp.yml
	$(MOBY) build --pull cases/test-ltp.yml

../artifacts/test-ltp.img.tar.gz: test-ltp.img.tar.gz | ../artifacts
	mv test-ltp.img.tar.gz ../artifacts

.PHONY: ltp
ltp: export CLOUDSDK_IMAGE_NAME?=test-ltp
ltp: $(LINUXKIT) test-ltp.img.tar.gz
	$(LINUXKIT) push gcp ../artifacts/test-ltp.img.tar.gz
	$(LINUXKIT) run gcp -skip-cleanup -machine n1-highcpu-4 $(CLOUDSDK_IMAGE_NAME) | tee test-ltp.log
	$(call check_test_log, test-ltp.log)

.PHONY: clean
clean:
	rm -rf bin *.log *-kernel *-cmdline *.img *.iso *.tar.gz *.qcow2 *.vhd *.vmx *.vmdk
