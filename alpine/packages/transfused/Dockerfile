FROM mobylinux/alpine-build-c:41e7634e137b022e6fef9d651166783966de6725

COPY . /transfused

WORKDIR /transfused

RUN cc -g -static -Wall -Werror -o transfused transfused.c transfused_log.c transfused_vsock.c

CMD ["tar", "cf", "-", "transfused"]
