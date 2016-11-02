FROM mobylinux/alpine-build-c:dfe78f283402fc2d6940c43763e2921491fb09fa

COPY . /9pmount-vsock

WORKDIR /9pmount-vsock

RUN cc -Wall -Werror 9pmount-vsock.c hvsock.c -lpthread -o 9pmount-vsock

CMD ["tar", "cf", "-", "9pmount-vsock"]
