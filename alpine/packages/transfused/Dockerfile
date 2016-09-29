FROM mobylinux/alpine-build-c:50a89f207d5fc98ce750d694077c75a31ab7937d

COPY . /transfused

WORKDIR /transfused

RUN cc -g -static -Wall -Werror -o transfused transfused.c transfused_log.c transfused_vsock.c

CMD ["tar", "cf", "-", "transfused"]
