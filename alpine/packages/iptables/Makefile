# This image is currently just tagged latest as non reproducible
IPTABLES_IMAGE=mobylinux/pinata-iptables@sha256:4dc27c9515c99dc80c8ec431de2c7db7d382425e5c1de13a560e6ec4c6a52b89

all: usr/local/sbin/iptables

usr/local/sbin/iptables:
	mkdir -p $(dir $@)
	docker run --rm --net=none --log-driver=none $(IPTABLES_IMAGE) | tar xf - -C $(dir $@)

clean:
	rm -rf usr

.DELETE_ON_ERROR:
