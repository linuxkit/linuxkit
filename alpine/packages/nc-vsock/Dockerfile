FROM mobylinux/alpine-build-c:50a89f207d5fc98ce750d694077c75a31ab7937d

COPY . /nc-vsock

WORKDIR /nc-vsock

RUN cc -Wall -Werror -o nc-vsock nc-vsock.c -luuid

CMD ["tar", "cf", "-", "nc-vsock"]
