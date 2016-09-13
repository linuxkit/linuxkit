FROM mobylinux/alpine-build-c:1ea690a7438cdd8a5965eaa034e0a0c521d9cb40

COPY . /transfused

WORKDIR /transfused

RUN cc -g -static -Wall -Werror -o transfused transfused.c transfused_log.c transfused_vsock.c

CMD ["tar", "cf", "-", "transfused"]
