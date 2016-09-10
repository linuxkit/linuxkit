all: usr/local/sbin/iptables

usr/local/sbin/iptables: Dockerfile main.ml
	mkdir -p usr/local/sbin
	docker build -t iptables:build .
	docker run --rm iptables:build cat /app/iptables > usr/local/sbin/iptables
	chmod 755 usr/local/sbin/iptables

clean:
	rm -rf usr
	docker images -q iptables:build | xargs docker rmi -f || true
