all: vsudd

vsudd: Dockerfile main.go
	docker build --build-arg GOOS=$(OS) --build-arg GOARCH=$(ARCH) -t vsudd:build .
	docker run --rm vsudd:build cat /go/bin/vsudd > vsudd
	chmod 755 vsudd

clean:
	rm -f vsudd
	docker images -q vsudd:build | xargs docker rmi -f
