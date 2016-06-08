all: llmnrd

llmnrd: Dockerfile src/*
	docker build -t llmnrd:build .
	docker run --rm llmnrd:build cat /llmnrd.tar.gz > llmnrd.tar.gz
	docker run --rm llmnrd:build cat /llmnrd/llmnrd > llmnrd
	chmod 755 llmnrd

clean:
	rm -f llmnrd llmnrd.tar.gz
	docker images -q llmnrd:build | xargs docker rmi -f || true
