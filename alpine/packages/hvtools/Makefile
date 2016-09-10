all: hv_fcopy_daemon hvtools.tar.gz

hv_fcopy_daemon: Dockerfile src/*
	tar cf - Dockerfile src | docker build -t hvtools:build -
	docker run --rm --net=none hvtools:build | tar xf -

hvtools.tar.gz: src/*
	tar czf hvtools.tar.gz src

clean:
	rm -f hv_fcopy_daemon hv_kvp_daemon hv_vss_daemon hvtools.tar.gz

.DELETE_ON_ERROR:
