.PHONY: license clean run
all: license

license: packages.lua Dockerfile license.sh
	docker build -t license .

run: license
	rm -rf output
	docker run -it -v /etc:/hostetc -v /usr:/hostusr -v /lib:/lib -v $(PWD)/output:/output license
	docker run -it --rm -v $(PWD):/licensing debian sh -c "cd /licensing/output && tar --numeric-owner -zcvf /licensing/License.tar.gz *"
	mv License.tar.gz ~/Desktop/

clean:
	rm -rf output
