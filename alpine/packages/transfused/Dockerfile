FROM mobylinux/alpine-build-c:f97e13b3a7339af5da5d620fc053c6a6c81b00a6

COPY . /transfused

WORKDIR /transfused

RUN cc -g -static -Wall -Werror -o transfused transfused.c transfused_log.c transfused_vsock.c

CMD ["tar", "cf", "-", "transfused"]
