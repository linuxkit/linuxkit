all: hvtools

hvtools: Dockerfile src/*
	docker build -t hvtools:build .
	docker run --rm hvtools:build cat /hvtools/hv_fcopy_daemon > hv_fcopy_daemon
	docker run --rm hvtools:build cat /hvtools/hv_kvp_daemon > hv_kvp_daemon
	docker run --rm hvtools:build cat /hvtools/hv_vss_daemon > hv_vss_daemon
	chmod 755 hv_fcopy_daemon hv_kvp_daemon hv_vss_daemon

clean:
	rm -f hv_fcopy_daemon hv_kvp_daemon hv_vss_daemon
	docker images -q hvtools:build | xargs -r docker rmi -f
