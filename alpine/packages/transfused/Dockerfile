FROM mobylinux/alpine-build-c:36aecb5cf4738737634140eec9abebe1f6559a39

COPY . /transfused

WORKDIR /transfused

RUN cc -g -static -Wall -Werror -o transfused transfused.c transfused_log.c transfused_vsock.c

CMD ["tar", "cf", "-", "transfused"]
