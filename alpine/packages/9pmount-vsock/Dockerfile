# Tag: 3da0b0ea0da2724232603094e67c75b41adab551
FROM mobylinux/alpine-build-c@sha256:0236d6599f6c8f7aa42829285e04202fbe99984df2818af0f5a453a59de8b090

COPY . /9pmount-vsock

WORKDIR /9pmount-vsock

RUN cc -Wall -Werror 9pmount-vsock.c hvsock.c -lpthread -o 9pmount-vsock

CMD ["tar", "cf", "-", "9pmount-vsock"]
