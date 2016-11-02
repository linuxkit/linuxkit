FROM mobylinux/alpine-build-c:dfe78f283402fc2d6940c43763e2921491fb09fa

COPY . /transfused

WORKDIR /transfused

RUN cc -g -static -Wall -Werror -o transfused transfused.c transfused_log.c transfused_vsock.c

CMD ["tar", "cf", "-", "transfused"]
