all: usr/local/sbin/iptables

usr/local/sbin/iptables: Dockerfile main.ml
	mkdir -p usr/local/sbin
	BUILD=$$( docker build -q . ) && [ -n "$$BUILD" ] && \
	docker run --rm $$BUILD | tar xf - -C usr/local/sbin && \
	docker rmi --no-prune $$BUILD

clean:
	rm -rf usr
	docker images -q iptables:build | xargs docker rmi -f || true
