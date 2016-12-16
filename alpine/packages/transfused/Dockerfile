# Tag: 3da0b0ea0da2724232603094e67c75b41adab551
FROM mobylinux/alpine-build-c@sha256:0236d6599f6c8f7aa42829285e04202fbe99984df2818af0f5a453a59de8b090

COPY . /transfused

WORKDIR /transfused

RUN cc -g -static -Wall -Werror -o transfused transfused.c transfused_log.c transfused_vsock.c

CMD ["tar", "cf", "-", "transfused"]
