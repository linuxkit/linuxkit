.PHONY: license clean run
all: license

license: packages.lua Dockerfile license.sh
	docker build -t license .

run: license
	rm -rf output
	docker run -it -v /etc:/hostetc -v /lib:/lib -v $(PWD)/output:/output license

clean:
	rm -rf output
