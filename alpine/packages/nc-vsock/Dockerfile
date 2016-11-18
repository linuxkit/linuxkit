FROM mobylinux/alpine-build-c:36aecb5cf4738737634140eec9abebe1f6559a39

COPY . /nc-vsock

WORKDIR /nc-vsock

RUN cc -Wall -Werror -o nc-vsock nc-vsock.c -luuid

CMD ["tar", "cf", "-", "nc-vsock"]
