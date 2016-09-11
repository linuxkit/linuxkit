FROM mobylinux/alpine-build-c:1ea690a7438cdd8a5965eaa034e0a0c521d9cb40

COPY . /9pmount-vsock

WORKDIR /9pmount-vsock

RUN cc -Wall -Werror 9pmount-vsock.c hvsock.c -lpthread -o 9pmount-vsock

CMD ["tar", "cf", "-", "9pmount-vsock"]
