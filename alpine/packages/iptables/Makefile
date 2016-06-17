all: iptables

iptables: Dockerfile main.ml
	docker build -t iptables:build .
	docker run --rm iptables:build cat /app/iptables > iptables
	chmod 755 iptables

clean:
	docker images -q iptables:build | xargs docker rmi -f || true
