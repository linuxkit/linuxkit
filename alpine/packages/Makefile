all:
	$(MAKE) -C proxy
	$(MAKE) -C diagnostics
	$(MAKE) -C transfused OS=linux
	$(MAKE) -C tap-vsockd
	$(MAKE) -C hvtools OS=linux
	$(MAKE) -C docker
	$(MAKE) -C nc-vsock OS=linux
	$(MAKE) -C vsudd
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
