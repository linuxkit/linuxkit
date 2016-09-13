FROM mobylinux/alpine-build-c:1ea690a7438cdd8a5965eaa034e0a0c521d9cb40

COPY src /hvtools/

WORKDIR /hvtools
RUN make

CMD ["tar", "cf", "-", "hv_fcopy_daemon", "hv_kvp_daemon", "hv_vss_daemon"]
