all: diagnostics-server

.PHONY: vendor
vendor:
	mkdir -p ./vendor
	cp -r ../go/vendor/* ./vendor/

diagnostics-server: Dockerfile main.go vendor
	docker build --build-arg GOOS=$(OS) --build-arg GOARCH=$(ARCH) -t diagnostics:build .
	docker run --rm diagnostics:build cat /go/bin/diagnostics > diagnostics-server
	chmod 755 diagnostics-server

clean:
	rm -rf diagnostics-server vendor
	docker images -q diagnostics:build | xargs docker rmi -f || true
