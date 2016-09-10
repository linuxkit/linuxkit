all: usr/local/sbin/iptables

usr/local/sbin/iptables: Dockerfile main.ml
	mkdir -p usr/local/sbin
	docker build -t iptables:build .
	docker run --rm iptables:build | tar xf - -C usr/local/sbin

clean:
	rm -rf usr
	docker images -q iptables:build | xargs docker rmi -f || true
