all: proxy

.PHONY: vendor
vendor:
	mkdir -p ./vendor
	cp -r ../go/vendor/* ./vendor/

proxy: Dockerfile main.go proxy.go vendor
	docker build --build-arg GOOS=$(OS) --build-arg GOARCH=$(ARCH) -t proxy:build .
	docker run --rm proxy:build cat /go/bin/proxy > proxy
	chmod 755 proxy

clean:
	rm -rf proxy vendor
	docker images -q proxy:build | xargs docker rmi -f || true
