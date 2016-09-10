all:
	$(MAKE) -C proxy
	$(MAKE) -C diagnostics OS=linux
	$(MAKE) -C transfused OS=linux
	$(MAKE) -C tap-vsockd OS=linux
	$(MAKE) -C hvtools OS=linux
	$(MAKE) -C docker OS=Linux
	$(MAKE) -C nc-vsock OS=linux
	$(MAKE) -C vsudd OS=linux
	$(MAKE) -C 9pmount-vsock OS=linux
	$(MAKE) -C iptables OS=linux

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
