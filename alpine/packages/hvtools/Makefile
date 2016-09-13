all: sbin/hv_fcopy_daemon usr/share/src/hvtools.tar.gz

sbin/hv_fcopy_daemon: Dockerfile src/*
	tar cf - Dockerfile src | docker build -t hvtools:build -
	docker run --rm --net=none hvtools:build | tar xf - -C sbin

usr/share/src/hvtools.tar.gz: src/*
	mkdir -p usr/share/src/
	tar czf usr/share/src/hvtools.tar.gz src

clean:
	rm -rf usr sbin/hv_fcopy_daemon sbin/hv_kvp_daemon sbin/hv_vss_daemon

.DELETE_ON_ERROR:
