GIT_VERSION := $(shell git describe --abbrev=6 --dirty --always --tags)

ifeq ($V, 1)
	VERBOSE =
else
	VERBOSE = @
endif

include config.mk

VMM_SRC := \
	src/vmm/x86.c \
	src/vmm/vmm.c \
	src/vmm/vmm_host.c \
	src/vmm/vmm_mem.c \
	src/vmm/vmm_lapic.c \
	src/vmm/vmm_instruction_emul.c \
	src/vmm/vmm_ioport.c \
	src/vmm/vmm_callout.c \
	src/vmm/vmm_stat.c \
	src/vmm/vmm_util.c \
	src/vmm/vmm_api.c \
	src/vmm/intel/vmx.c \
	src/vmm/intel/vmx_msr.c \
	src/vmm/intel/vmcs.c \
	src/vmm/io/vatpic.c \
	src/vmm/io/vatpit.c \
	src/vmm/io/vhpet.c \
	src/vmm/io/vioapic.c \
	src/vmm/io/vlapic.c \
	src/vmm/io/vpmtmr.c \
	src/vmm/io/vrtc.c

XHYVE_SRC := \
	src/acpitbl.c \
	src/atkbdc.c \
	src/block_if.c \
	src/consport.c \
	src/dbgport.c \
	src/inout.c \
	src/ioapic.c \
	src/md5c.c \
	src/mem.c \
	src/mevent.c \
	src/mptbl.c \
	src/pci_ahci.c \
	src/pci_emul.c \
	src/pci_hostbridge.c \
	src/pci_irq.c \
	src/pci_lpc.c \
	src/pci_uart.c \
	src/pci_virtio_block.c \
	src/pci_virtio_net_tap.c \
	src/pci_virtio_net_vmnet.c \
	src/pci_virtio_rnd.c \
	src/pci_virtio_9p.c \
	src/pm.c \
	src/post.c \
	src/rtc.c \
	src/smbiostbl.c \
	src/task_switch.c \
	src/uart_emul.c \
	src/xhyve.c \
	src/virtio.c \
	src/xmsr.c

FIRMWARE_SRC := \
	src/firmware/kexec.c \
	src/firmware/fbsd.c

SRC := \
	$(VMM_SRC) \
	$(XHYVE_SRC) \
	$(FIRMWARE_SRC)

OBJ := $(SRC:src/%.c=build/%.o)
DEP := $(OBJ:%.o=%.d)
INC := -Iinclude

CFLAGS += -DVERSION=\"$(GIT_VERSION)\"

TARGET = build/xhyve

all: $(TARGET) | build

.PHONY: clean all
.SUFFIXES:

-include $(DEP)

build:
	@mkdir -p build

build/%.o: src/%.c
	@echo cc $<
	@mkdir -p $(dir $@)
	$(VERBOSE) $(ENV) $(CC) $(CFLAGS) $(INC) $(DEF) -MMD -MT $@ -MF build/$*.d -o $@ -c $<

$(TARGET).sym: $(OBJ)
	@echo ld $(notdir $@)
	$(VERBOSE) $(ENV) $(LD) $(LDFLAGS) -Xlinker $(TARGET).lto.o -o $@ $(OBJ)
	@echo dsym $(notdir $(TARGET).dSYM)
	$(VERBOSE) $(ENV) $(DSYM) $@ -o $(TARGET).dSYM

$(TARGET): $(TARGET).sym
	@echo strip $(notdir $@)
	$(VERBOSE) $(ENV) $(STRIP) $(TARGET).sym -o $@

disk.img:
	dd if=/dev/zero of=disk.img bs=4096 count=10000k

run: $(TARGET) disk.img
	./xhyverun.sh || true

clean:
	@rm -rf build
	@rm -f disk.img
