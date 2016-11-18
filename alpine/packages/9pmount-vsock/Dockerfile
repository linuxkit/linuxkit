FROM mobylinux/alpine-build-c:36aecb5cf4738737634140eec9abebe1f6559a39

COPY . /9pmount-vsock

WORKDIR /9pmount-vsock

RUN cc -Wall -Werror 9pmount-vsock.c hvsock.c -lpthread -o 9pmount-vsock

CMD ["tar", "cf", "-", "9pmount-vsock"]
