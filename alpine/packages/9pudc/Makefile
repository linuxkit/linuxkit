all: 9pudc

9pudc: Dockerfile main.go
	docker build -t 9pudc:test .
	docker run 9pudc:test cat /go/bin/9pudc > 9pudc

clean:
	rm -f 9pudc
