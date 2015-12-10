all: mdnstool

mdnstool: Dockerfile mdnstool.go mdnsmon/mdnsmon.go
	docker build -t mdnstool:test .
	docker run mdnstool:test cat /go/bin/mdnstool > mdnstool
	chmod 555 mdnstool

clean:
	rm -f mdnstool
