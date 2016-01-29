all: llmnrd

llmnrd: Dockerfile src/*
	docker build -t llmnrd:build .
	docker run llmnrd:build cat /llmnrd/llmnrd > llmnrd
	chmod 755 llmnrd

clean:
	rm -f llmnrd
	docker images -q llmnrd:build | xargs docker rmi -f
