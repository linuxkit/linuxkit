FROM mobylinux/alpine-build-c:1b53976210ba819823eda5b1d0d3ea337772e97b

COPY src /hvtools/

WORKDIR /hvtools
RUN make

CMD ["tar", "cf", "-", "hv_fcopy_daemon", "hv_kvp_daemon", "hv_vss_daemon"]
