# Tag: 36aecb5cf4738737634140eec9abebe1f6559a39
FROM mobylinux/alpine-build-c@sha256:d66b9625abc831f28f8c584991a9cb6975e85d3bb3d3768474b592f1cf32a3a6

COPY . /9pmount-vsock

WORKDIR /9pmount-vsock

RUN cc -Wall -Werror 9pmount-vsock.c hvsock.c -lpthread -o 9pmount-vsock

CMD ["tar", "cf", "-", "9pmount-vsock"]
