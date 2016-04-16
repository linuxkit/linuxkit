all: linuxx64.efi.stub

linuxx64.efi.stub: Dockerfile gummiboot/*
	docker build -t gummiboot:build .
	docker run --rm gummiboot:build cat /gummiboot.tar.gz > gummiboot.tar.gz
	docker run --rm gummiboot:build \
		cat /gummiboot/linuxx64.efi.stub > linuxx64.efi.stub

clean:
	rm -f linuxx64.efi.stub gummiboot.tar.gz
	docker images -q gummiboot:build | xargs docker rmi -f
