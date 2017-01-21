# This image is currently just tagged latest as non reproducible
IPTABLES_IMAGE=mobylinux/pinata-iptables@sha256:c94aa6902e858712058d0550320564ecb03d1f8f6370f91cdbc3f18b87662411

all: usr/local/sbin/iptables

usr/local/sbin/iptables:
	mkdir -p $(dir $@)
	docker run --rm --net=none --log-driver=none $(IPTABLES_IMAGE) | tar xf - -C $(dir $@)

clean:
	rm -rf usr

.DELETE_ON_ERROR:
