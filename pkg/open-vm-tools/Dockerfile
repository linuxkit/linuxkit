FROM linuxkit/alpine:b3028815dbf119404ab68a25acfcf84c585ab432 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    open-vm-tools \
    open-vm-tools-guestinfo \
    open-vm-tools-deploypkg \
    open-vm-tools-static \
    open-vm-tools-vmbackup 

# Remove apk residuals
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
COPY scripts /etc/vmware-tools/scripts
CMD ["/usr/bin/vmtoolsd"]
