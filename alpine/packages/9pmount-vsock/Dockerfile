FROM mobylinux/alpine-build-c:7303e33e9dcd5276b8bb5269644a9bf3354008c8

COPY . /9pmount-vsock

WORKDIR /9pmount-vsock

RUN cc -Wall -Werror 9pmount-vsock.c hvsock.c -lpthread -o 9pmount-vsock

CMD ["tar", "cf", "-", "9pmount-vsock"]
