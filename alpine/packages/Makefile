all:
	$(MAKE) -C proxy
	$(MAKE) -C diagnostics
	$(MAKE) -C transfused
	$(MAKE) -C tap-vsockd
	$(MAKE) -C hvtools
	$(MAKE) -C docker
	$(MAKE) -C nc-vsock
	$(MAKE) -C vsudd
	$(MAKE) -C 9pmount-vsock
	$(MAKE) -C iptables

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
