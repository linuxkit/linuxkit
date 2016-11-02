FROM mobylinux/alpine-build-c:dfe78f283402fc2d6940c43763e2921491fb09fa

COPY . /nc-vsock

WORKDIR /nc-vsock

RUN cc -Wall -Werror -o nc-vsock nc-vsock.c -luuid

CMD ["tar", "cf", "-", "nc-vsock"]
