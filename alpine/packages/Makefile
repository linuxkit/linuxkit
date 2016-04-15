all:
	$(MAKE) -C proxy OS=linux
	$(MAKE) -C diagnostics OS=linux
	$(MAKE) -C transfused OS=linux
	$(MAKE) -C mdnstool OS=linux
	$(MAKE) -C hupper OS=linux
	$(MAKE) -C hvtools OS=linux
	$(MAKE) -C docker OS=Linux
	$(MAKE) -C nc-vsock OS=linux
	$(MAKE) -C vsudd OS=linux

arm:
	$(MAKE) -C proxy OS=linux ARCH=arm
	$(MAKE) -C diagnostics OS=linux ARCH=arm
	$(MAKE) -C transfused OS=linux ARCH=arm
	$(MAKE) -C mdnstool OS=linux ARCH=arm
	$(MAKE) -C hupper OS=linux ARCH=arm
	$(MAKE) -C docker OS=Linux ARCH=arm
	$(MAKE) -C nc-vsock OS=linux ARCH=arm
	$(MAKE) -C vsudd OS=linux ARCH=arm

clean:
	$(MAKE) -C proxy clean
	$(MAKE) -C diagnostics clean
	$(MAKE) -C transfused clean
	$(MAKE) -C mdnstool clean
	$(MAKE) -C docker clean
	$(MAKE) -C hupper clean
	$(MAKE) -C hvtools clean
	$(MAKE) -C nc-vsock clean
	$(MAKE) -C vsudd clean
