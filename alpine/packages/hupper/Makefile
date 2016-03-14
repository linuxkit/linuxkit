all: hupper

hupper: Dockerfile main.go
	docker build --build-arg GOOS=linux --build-arg GOARCH=amd64 -t hupper:build .
	docker run --rm hupper:build cat /go/bin/hupper > hupper
	chmod 755 hupper

clean:
	rm -f hupper
	docker images -q hupper:build | xargs docker rmi -f
