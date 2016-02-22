all: mdnstool

mdnstool: Dockerfile mdnstool.go mdnsmon/mdnsmon.go
	docker build --build-arg GOOS=$(OS) --build-arg GOARCH=$(ARCH) -t mdnstool:build .
	docker run --rm mdnstool:build cat /go/bin/mdnstool > mdnstool
	chmod 755 mdnstool

clean:
	rm -f mdnstool
	docker images -q mdnstool:build | xargs -r docker rmi -f
