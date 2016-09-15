DEPS=proxy diagnostics transfused tap-vsockd hvtools docker nc-vsock vsudd 9pmount-vsock iptables random
.PHONY: clean $(DEPS)

default: $(DEPS)

$(DEPS):
	$(MAKE) -C $@

clean:
	for f in $(DEPS); do $(MAKE) -C $$f clean; done
