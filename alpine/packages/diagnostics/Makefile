all: diagnostics-server

diagnostics-server: Dockerfile main.go
	docker build --build-arg GOOS=$(OS) --build-arg GOARCH=$(ARCH) -t diagnostics:build .
	docker run --rm diagnostics:build cat /go/bin/diagnostics > diagnostics-server
	chmod 755 diagnostics-server

clean:
	rm -f diagnostics-server
	docker images -q diagnostics:build | xargs docker rmi -f
