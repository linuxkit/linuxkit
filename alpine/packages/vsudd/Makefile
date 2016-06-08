all: vsudd

.PHONY: vendor
vendor:
	mkdir -p ./vendor
	cp -r ../go/vendor/* ./vendor/

vsudd: Dockerfile main.go vendor
	docker build --build-arg GOOS=$(OS) --build-arg GOARCH=$(ARCH) -t vsudd:build .
	docker run --rm vsudd:build cat /go/bin/vsudd > vsudd
	chmod 755 vsudd

clean:
	rm -rf vsudd vendor
	docker images -q vsudd:build | xargs docker rmi -f || true
