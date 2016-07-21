all:
	$(MAKE) -C proxy OS=linux
	$(MAKE) -C diagnostics OS=linux
	$(MAKE) -C transfused OS=linux
	$(MAKE) -C tap-vsockd OS=linux
	$(MAKE) -C hvtools OS=linux
	$(MAKE) -C docker OS=Linux
	$(MAKE) -C nc-vsock OS=linux
	$(MAKE) -C vsudd OS=linux
	$(MAKE) -C 9pmount-vsock OS=linux
	$(MAKE) -C iptables OS=linux

arm:
	$(MAKE) -C transfused OS=linux ARCH=arm
	$(MAKE) -C docker arm OS=Linux ARCH=arm
	# Not cross building at present (C code)
	# $(MAKE) -C tap-vsockd OS=linux ARCH=arm
	# $(MAKE) -C diagnostics OS=linux ARCH=arm
	# $(MAKE) -C proxy OS=linux ARCH=arm
	# $(MAKE) -C nc-vsock OS=linux ARCH=arm
	# $(MAKE) -C vsudd OS=linux ARCH=arm

clean:
	$(MAKE) -C proxy clean
	$(MAKE) -C diagnostics clean
	$(MAKE) -C transfused clean
	$(MAKE) -C tap-vsockd clean
	$(MAKE) -C docker clean
	$(MAKE) -C hvtools clean
	$(MAKE) -C nc-vsock clean
	$(MAKE) -C vsudd clean
	$(MAKE) -C 9pmount-vsock clean
	$(MAKE) -C iptables clean
