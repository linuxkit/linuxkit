BASE=ocaml/opam:alpine
IMAGE=pinata-iptables

# OCaml builds are non deterministic so do not generate a hash

default: Dockerfile main.ml
	docker pull $(BASE)
	BUILD=$$( docker build -q . ) && \
	[ -n "$$BUILD" ] && \
	echo "Built $$BUILD" && \
	docker tag $$BUILD mobylinux/$(IMAGE):latest
	docker push mobylinux/$(IMAGE):latest
