.PHONY: extract
extract: iid
	rm -rf ./build
	mkdir -p ./build
	docker run --rm $(shell cat iid) | tar xf - -C ./build
	rm iid

iid: Makefile Dockerfile
	docker build --no-cache --iidfile iid .
