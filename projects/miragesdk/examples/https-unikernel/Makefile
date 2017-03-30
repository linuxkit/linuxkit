.PHONY: all docker docker-run clean

all: www-data
	jbuilder build --dev src/main.exe src/store_main.exe src/http_main.exe src/tls_main.exe

docker:
	docker build -t https-unikernel .

docker-run: tls-secrets/server.crt
	docker run --rm -it -v ${PWD}/tls-secrets:/home/opam/src/tls-secrets -p 8443:8443 https-unikernel bash -c 'reset; screen'

clean:
	rm -rf _build

tls-secrets/server.key:
	@echo Generating server key...
	[ -d tls-secrets ] || mkdir -p tls-secrets
	openssl genrsa -out $@ 4096

tls-secrets/server.crt: tls-secrets/server.key
	@echo ">>> Generating server X.509 certificate."
	@echo ">>> Enter the server's full hostname as the 'Common Name' (e.g. demo.mynet)."
	@echo ">>> Everything else can be left blank."
	@echo
	@openssl req -new -x509 -key $< -out $@ -days 10000

www-data:
	git init www-data
	echo '<p>It works!</p><p>Powered by Irmin.</p>' > www-data/index.html
	git -C www-data add index.html
	git -C www-data commit --author 'Test <test@example.com>' -m 'Initial state'
