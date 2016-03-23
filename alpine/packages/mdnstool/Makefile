all: mdnstool

mdnstool: Dockerfile mdnstool.go mdnsmon/mdnsmon.go
	docker build --build-arg GOOS=$(OS) --build-arg GOARCH=$(ARCH) -t mdnstool:build .
	docker run --rm mdnstool:build cat /go/bin/mdnstool > mdnstool
	chmod 755 mdnstool

vendor:
	docker run -v $(PWD):/go/src -it justincormack/gvt sh -c 'cd /go/src; gvt fetch github.com/hashicorp/mdns'

clean:
	rm -f mdnstool
	docker images -q mdnstool:build | xargs docker rmi -f
