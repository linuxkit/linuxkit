FROM mobylinux/alpine-build-c:f97e13b3a7339af5da5d620fc053c6a6c81b00a6

COPY . /9pmount-vsock

WORKDIR /9pmount-vsock

RUN cc -Wall -Werror 9pmount-vsock.c hvsock.c -lpthread -o 9pmount-vsock

CMD ["tar", "cf", "-", "9pmount-vsock"]
