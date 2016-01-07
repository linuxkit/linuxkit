all: 9pudc

9pudc: Dockerfile main.go
	docker build --build-arg GOOS=$(GOOS) --build-arg GOARCH=$(GOARCH) -t 9pudc:build .
	docker run 9pudc:build cat /go/bin/9pudc > 9pudc
	chmod 755 9pudc

clean:
	rm -f 9pudc
	docker images -q 9pudc:build | xargs docker rmi -f
