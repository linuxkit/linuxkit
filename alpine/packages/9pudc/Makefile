all: 9pudc

9pudc: Dockerfile main.go
	docker build --build-arg GOOS=linux --build-arg GOARCH=amd64 -t 9pudc:build .
	docker run --rm 9pudc:build cat /go/bin/9pudc > 9pudc
	chmod 755 9pudc

clean:
	rm -f 9pudc
	docker images -q 9pudc:build | xargs docker rmi -f
