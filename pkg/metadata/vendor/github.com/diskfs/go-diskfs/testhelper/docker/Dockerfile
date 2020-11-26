FROM alpine:3.8

# just install the tools we need
RUN apk --update add dosfstools mtools sgdisk sfdisk gptfdisk p7zip cdrkit

COPY entrypoint.sh /

ENTRYPOINT ["/entrypoint.sh"]
