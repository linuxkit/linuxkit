# Tag: 3da0b0ea0da2724232603094e67c75b41adab551
FROM mobylinux/alpine-build-c@sha256:0236d6599f6c8f7aa42829285e04202fbe99984df2818af0f5a453a59de8b090

COPY . /nc-vsock

WORKDIR /nc-vsock

RUN cc -Wall -Werror -o nc-vsock nc-vsock.c -luuid

CMD ["tar", "cf", "-", "nc-vsock"]
