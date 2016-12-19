DEPS=proxy diagnostics transfused tap-vsockd docker nc-vsock vsudd 9pmount-vsock iptables
.PHONY: clean $(DEPS)

default: $(DEPS)

$(DEPS): lint
	$(MAKE) -C $@

lint:
	# gofmt
	@test -z "$$(gofmt -s -l .| grep -v .pb. | grep -v vendor/ | tee /dev/stderr)"
	# govet
	@test -z "$$(go tool vet -printf=false . 2>&1 | grep -v vendor/ | tee /dev/stderr)"

clean:
	for f in $(DEPS); do $(MAKE) -C $$f clean; done
