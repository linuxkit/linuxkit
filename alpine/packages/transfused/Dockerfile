FROM mobylinux/alpine-build-c:1b53976210ba819823eda5b1d0d3ea337772e97b

COPY . /transfused

WORKDIR /transfused

RUN cc -g -static -Wall -Werror -o transfused transfused.c transfused_log.c transfused_vsock.c

CMD ["tar", "cf", "-", "transfused"]
