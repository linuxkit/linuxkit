FROM mobylinux/alpine-build-c:7303e33e9dcd5276b8bb5269644a9bf3354008c8

COPY . /transfused

WORKDIR /transfused

RUN cc -g -static -Wall -Werror -o transfused transfused.c transfused_log.c transfused_vsock.c

CMD ["tar", "cf", "-", "transfused"]
