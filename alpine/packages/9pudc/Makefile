all: 9pudc

9pudc: Dockerfile main.go
	docker build --build-arg GOOS=$(GOOS) --build-arg GOARCH=$(GOARCH) -t 9pudc:test .
	docker run 9pudc:test cat /go/bin/9pudc > 9pudc
	chmod 755 9pudc

clean:
	rm -f 9pudc
