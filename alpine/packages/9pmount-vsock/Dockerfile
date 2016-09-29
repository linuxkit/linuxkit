FROM mobylinux/alpine-build-c:50a89f207d5fc98ce750d694077c75a31ab7937d

COPY . /9pmount-vsock

WORKDIR /9pmount-vsock

RUN cc -Wall -Werror 9pmount-vsock.c hvsock.c -lpthread -o 9pmount-vsock

CMD ["tar", "cf", "-", "9pmount-vsock"]
