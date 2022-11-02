PUIPUI_LINUX_VERSION := 0.0.1
ARCH := $(shell uname -m)
KERNEL_ARCH := $(shell echo $(ARCH) | sed -e s/arm64/aarch64/)
KERNEL_TAR := puipui_linux_v$(PUIPUI_LINUX_VERSION)_$(KERNEL_ARCH).tar.gz
KERNEL_DOWNLOAD_URL := https://github.com/Code-Hex/puipui-linux/releases/download/v$(PUIPUI_LINUX_VERSION)/$(KERNEL_TAR)

.PHONY: fmt
fmt:
	@ls | grep -E '\.(h|m)$$' | xargs clang-format -i --verbose

.PHONY: test
test:
	go test -exec "go run $(PWD)/cmd/codesign" ./... -timeout 60s -v

.PHONY: download_kernel
download_kernel:
	curl --output-dir testdata -LO $(KERNEL_DOWNLOAD_URL)
	@tar xvf testdata/$(KERNEL_TAR) -C testdata
ifeq ($(ARCH),arm64)
	@gunzip -f testdata/Image.gz
else
	@mv testdata/bzImage testdata/Image
endif

.PHONY: install/stringer
install/stringer:
	@go install golang.org/x/tools/cmd/stringer@latest

.PHONY: clean
clean:
	@rm testdata/{Image,initramfs.cpio.gz,*.tar.gz}