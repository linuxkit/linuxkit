all: proxy

proxy: Dockerfile main.go proxy.go
	docker build --build-arg GOOS=$(OS) --build-arg GOARCH=$(ARCH) -t proxy:build .
	docker run --rm proxy:build cat /go/bin/proxy > proxy
	chmod 755 proxy

clean:
	rm -f proxy
	docker images -q proxy:build | xargs docker rmi -f
